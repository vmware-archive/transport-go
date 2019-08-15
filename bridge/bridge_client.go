package bridge

import "C"
import (
    "bifrost/bus"
    "bufio"
    "bytes"
    "errors"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/go-stomp/stomp/frame"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "sync"
)

type BridgeClient struct {
    WSc              *websocket.Conn // WebSocket connection
    TCPc             *stomp.Conn     // STOMP TCP Connection
    ConnectedChan    chan bool
    disconnectedChan chan bool
    connected        bool
    inboundChan      chan *frame.Frame
    stompConnected   bool
    Subscriptions    map[string]*BridgeClientSub
    logger           *log.Logger
    lock             sync.Mutex
}

func NewBridgeWsClient() *BridgeClient {
    return newBridgeWsClient()
}

func newBridgeWsClient() *BridgeClient {
    l := log.New(os.Stderr, "WebSocket Client: ", 2)
    return &BridgeClient{
        WSc:              nil,
        TCPc:             nil,
        stompConnected:   false,
        connected:        false,
        logger:           l,
        lock:             sync.Mutex{},
        Subscriptions:    make(map[string]*BridgeClientSub),
        ConnectedChan:    make(chan bool),
        disconnectedChan: make(chan bool),
        inboundChan:      make(chan *frame.Frame)}
}

func (ws *BridgeClient) Connect(url *url.URL, headers http.Header) error {
    ws.lock.Lock()
    defer ws.lock.Unlock()
    ws.logger.Printf("connecting to fabric endpoint over %s", url.String())

    c, _, err := websocket.DefaultDialer.Dial(url.String(), headers)
    if err != nil {
        return err
    }
    ws.WSc = c
    go ws.handleCommands()
    go ws.listenSocket()

    ws.SendFrame(frame.New(frame.CONNECT, frame.AcceptVersion, string(stomp.V12)))

    <-ws.ConnectedChan

    return nil
}

func (ws *BridgeClient) Disconnect() error {
    if ws.WSc != nil {
        defer ws.WSc.Close()
        ws.disconnectedChan <- true
    } else {
        return fmt.Errorf("cannot disconnect, no connection defined")
    }
    return nil
}

func (ws *BridgeClient) Subscribe(destination string) *BridgeClientSub {
    ws.lock.Lock()
    defer ws.lock.Unlock()
    id := uuid.New()
    s := &BridgeClientSub{
        C:           make(chan *bus.Message),
        Id:          &id,
        Client:      ws,
        Destination: destination,
        subscribed:  true}

    ws.Subscriptions[destination] = s

    subscribeFrame := frame.New(frame.SUBSCRIBE,
        frame.Id, id.String(),
        frame.Destination, destination,
        frame.Ack, stomp.AckAuto.String())

    go ws.SendFrame(subscribeFrame)
    return s
}

func (ws *BridgeClient) Send(destination string, payload []byte) {
    ws.lock.Lock()
    defer ws.lock.Unlock()
    sendFrame := frame.New(frame.SEND,
        frame.Destination, destination,
        frame.ContentLength, strconv.Itoa(len(payload)),
        frame.ContentType, "application/json")

    sendFrame.Body = payload
    go ws.SendFrame(sendFrame)

}

func (ws *BridgeClient) SendFrame(f *frame.Frame) {
    var b bytes.Buffer
    br := bufio.NewWriter(&b)
    sw := frame.NewWriter(br)
    sw.Write(f)
    w, _ := ws.WSc.NextWriter(websocket.TextMessage)
    w.Write(b.Bytes())
    defer w.Close()
}

func (ws *BridgeClient) listenSocket() {
    for {
        _, p, err := ws.WSc.ReadMessage()
        b := bytes.NewReader(p)
        sr := frame.NewReader(b)
        f, _ := sr.Read()

        if err != nil {
            break // socket can't be read anymore, exit.
        }
        if f != nil {
            ws.logger.Printf("Received STOMP Frame: %s\n", f.Command)
            ws.inboundChan <- f
        }
    }
}

func (ws *BridgeClient) handleCommands() {
    for {
        select {
        case f := <-ws.inboundChan:
            switch f.Command {
            case frame.CONNECTED:
                ws.logger.Printf("STOMP Client connected")
                ws.stompConnected = true
                ws.connected = true
                ws.ConnectedChan <- true

            case frame.MESSAGE:
                ws.logger.Printf("STOMP Message received")

                for _, sub := range ws.Subscriptions {
                    if sub.Destination == f.Header.Get(frame.Destination) {
                        c := &bus.MessageConfig{Payload: f.Body, Destination: sub.Destination}
                        if sub.subscribed {
                            sub.C <- bus.GenerateResponse(c)
                        }
                    }
                }

            case frame.ERROR:
                ws.logger.Printf("STOMP Error received")

                for _, sub := range ws.Subscriptions {
                    if sub.Destination == f.Header.Get(frame.Destination) {
                        c := &bus.MessageConfig{Payload: f.Body, Err: errors.New("STOMP Error " + string(f.Body))}
                        sub.E <- bus.GenerateError(c)
                    }
                }
            }
        case <-ws.disconnectedChan:
            break
        }

    }
}

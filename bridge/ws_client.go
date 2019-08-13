package bridge

import "C"
import (
    "bufio"
    "bytes"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/go-stomp/stomp/frame"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "log"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "strconv"
)

type BridgeWsClient struct {
    C              *websocket.Conn
    ConnectedChan  chan bool
    interrupt      chan os.Signal
    inboundChan    chan *frame.Frame
    stompConnected bool
    Subscriptions  map[string]*WsClientSub
    logger         *log.Logger
}

type WsClientSub struct {
    C           chan *frame.Frame   // MESSAGE payloads
    E           chan *frame.Frame   // ERROR payloads.
    Id          *uuid.UUID
    Destination string
    wsClient    *BridgeWsClient
}

func NewBridgeWsClient() *BridgeWsClient {
    return newBridgeWsClient()
}

func newBridgeWsClient() *BridgeWsClient {
    l := log.New(os.Stderr, "WebSocket Client: ", 2)
    return &BridgeWsClient{
        C:              nil,
        stompConnected: false,
        logger:         l,
        Subscriptions:  make(map[string]*WsClientSub),
        ConnectedChan:  make(chan bool),
        inboundChan:    make(chan *frame.Frame),
        interrupt:      make(chan os.Signal, 1)}
}

func (ws *BridgeWsClient) Connect(url *url.URL, headers http.Header) error {

    signal.Notify(ws.interrupt, os.Interrupt)

    ws.logger.Printf("connecting to fabric endpoint over %s", url.String())

    c, _, err := websocket.DefaultDialer.Dial(url.String(), headers)
    if err != nil {
        ws.logger.Fatal("cannot connect to endpoint:", err)
        return err
    }
    ws.C = c
    go ws.handleCommands()
    go ws.listenSocket()

    ws.SendFrame(frame.New(frame.CONNECT, frame.AcceptVersion, string(stomp.V12)))

    <-ws.ConnectedChan

    return nil
}

func (ws *BridgeWsClient) Disconnect() error {
    if ws.C != nil {
        defer ws.C.Close()
    } else {
        return fmt.Errorf("cannot disconnect, no connection defined")
    }
    return nil
}

func (ws *BridgeWsClient) Subscribe(destination string) *WsClientSub {
    id := uuid.New()
    s := &WsClientSub{
        C:           make(chan *frame.Frame),
        Id:          &id,
        wsClient:    ws,
        Destination: destination}

    ws.Subscriptions[destination] = s

    subscribeFrame := frame.New(frame.SUBSCRIBE,
        frame.Id, id.String(),
        frame.Destination, destination,
        frame.Ack, stomp.AckAuto.String())

    ws.SendFrame(subscribeFrame)
    return s
}

func (cs *WsClientSub) Unsubscribe() {

    unsubscribeFrame := frame.New(frame.UNSUBSCRIBE,
        frame.Id, cs.Id.String(),
        frame.Destination, cs.Destination,
        frame.Ack, stomp.AckAuto.String())

    cs.wsClient.SendFrame(unsubscribeFrame)

}

func (ws *BridgeWsClient) Send(destination string, payload []byte) {

    sendFrame := frame.New(frame.SEND,
        frame.Destination, destination,
        frame.ContentLength, strconv.Itoa(len(payload)),
        frame.ContentType, "application/json")

    sendFrame.Body = payload
    ws.SendFrame(sendFrame)

}

func (ws *BridgeWsClient) SendFrame(f *frame.Frame) {
    var b bytes.Buffer
    br := bufio.NewWriter(&b)
    sw := frame.NewWriter(br)
    sw.Write(f)
    w, _ := ws.C.NextWriter(websocket.TextMessage)
    w.Write(b.Bytes())
    defer w.Close()
}

func (ws *BridgeWsClient) listenSocket() {
    for {
        _, p, err := ws.C.ReadMessage()

        b := bytes.NewReader(p)
        sr := frame.NewReader(b)
        f, _ := sr.Read()

        if err != nil {
            log.Println(err)
            return
        }
        ws.logger.Printf("Received STOMP Frame: %s\n", f.Command)
        ws.inboundChan <- f
    }
}

func (ws *BridgeWsClient) handleCommands() {
    for {
        f := <-ws.inboundChan
        switch f.Command {
        case frame.CONNECTED:
            ws.logger.Printf("STOMP Client connected")
            ws.stompConnected = true
            ws.ConnectedChan <- true

        case frame.MESSAGE:
            ws.logger.Printf("STOMP Message received")

            for _, sub := range ws.Subscriptions {
                if sub.Destination == f.Header.Get(frame.Destination) {
                    sub.C <- f
                }
            }

        case frame.ERROR:
            ws.logger.Printf("STOMP Error received")

            for _, sub := range ws.Subscriptions {
                if sub.Destination == f.Header.Get(frame.Destination) {
                    sub.E <- f
                }
            }
        }

    }
}

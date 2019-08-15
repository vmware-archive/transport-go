// Copyright 2019 VMware Inc.

package bridge

import "C"
import (
    "bifrost/model"
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

// Bridge client encapsulates all subscriptions and io to and from brokers.
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

// Create a new WebSocket client.
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

// Connect to broker endpoint.
func (ws *BridgeClient) Connect(url *url.URL, headers http.Header) error {
    ws.lock.Lock()
    defer ws.lock.Unlock()
    ws.logger.Printf("connecting to fabric endpoint over %s", url.String())

    c, _, err := websocket.DefaultDialer.Dial(url.String(), headers)
    if err != nil {
        return err
    }
    ws.WSc = c

    // handle incoming STOMP frames.
    go ws.handleIncomingSTOMPFrames()

    // go listen to the websocket
    go ws.listenSocket()

    // send connect frame.
    ws.SendFrame(frame.New(frame.CONNECT, frame.AcceptVersion, string(stomp.V12)))

    // wait to be connected
    <-ws.ConnectedChan
    return nil
}

// Disconnect from broker endpoint
func (ws *BridgeClient) Disconnect() error {
    if ws.WSc != nil {
        defer ws.WSc.Close()
        ws.disconnectedChan <- true
    } else {
        return fmt.Errorf("cannot disconnect, no connection defined")
    }
    return nil
}

// Subscribe to destination
func (ws *BridgeClient) Subscribe(destination string) *BridgeClientSub {
    ws.lock.Lock()
    defer ws.lock.Unlock()
    id := uuid.New()
    s := &BridgeClientSub{
        C:           make(chan *model.Message),
        Id:          &id,
        Client:      ws,
        Destination: destination,
        subscribed:  true}

    ws.Subscriptions[destination] = s

    // create subscription frame.
    subscribeFrame := frame.New(frame.SUBSCRIBE,
        frame.Id, id.String(),
        frame.Destination, destination,
        frame.Ack, stomp.AckAuto.String())


    // send subscription frame.
    go ws.SendFrame(subscribeFrame)
    return s
}

// send a payload to a destination
func (ws *BridgeClient) Send(destination string, payload []byte) {
    ws.lock.Lock()
    defer ws.lock.Unlock()

    // create send frame.
    sendFrame := frame.New(frame.SEND,
        frame.Destination, destination,
        frame.ContentLength, strconv.Itoa(len(payload)),
        frame.ContentType, "application/json")

    // add payload
    sendFrame.Body = payload

    // send frame
    go ws.SendFrame(sendFrame)

}

// send a STOMP frame down the WebSocket
func (ws *BridgeClient) SendFrame(f *frame.Frame) {
    var b bytes.Buffer
    br := bufio.NewWriter(&b)
    sw := frame.NewWriter(br)

    // write frame to buffer
    sw.Write(f)
    w, _ := ws.WSc.NextWriter(websocket.TextMessage)
    defer w.Close()

    // write buffer bytes to websocket
    w.Write(b.Bytes())
}

func (ws *BridgeClient) listenSocket() {
    for {
        // read each incoming message from websocket
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

func (ws *BridgeClient) handleIncomingSTOMPFrames() {
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
                        c := &model.MessageConfig{Payload: f.Body, Destination: sub.Destination}
                        if sub.subscribed {
                            sub.C <- model.GenerateResponse(c)
                        }
                    }
                }

            case frame.ERROR:
                ws.logger.Printf("STOMP ErrorDir received")

                for _, sub := range ws.Subscriptions {
                    if sub.Destination == f.Header.Get(frame.Destination) {
                        c := &model.MessageConfig{Payload: f.Body, Err: errors.New("STOMP ErrorDir " + string(f.Body))}
                        sub.E <- model.GenerateError(c)
                    }
                }
            }
        case <-ws.disconnectedChan:
            break
        }

    }
}

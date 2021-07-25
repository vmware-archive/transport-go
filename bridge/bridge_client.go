// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
    "bufio"
    "bytes"
    "errors"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/go-stomp/stomp/frame"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "github.com/vmware/transport-go/model"
    "log"
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
    sendLock         sync.Mutex
}

// Create a new WebSocket client.
func NewBridgeWsClient(enableLogging bool) *BridgeClient {
    return newBridgeWsClient(enableLogging)
}

func newBridgeWsClient(enableLogging bool) *BridgeClient {
    var l *log.Logger = nil
    if enableLogging {
        l = log.New(os.Stderr, "WebSocket Client: ", 2)
    }
    return &BridgeClient{
        WSc:              nil,
        TCPc:             nil,
        stompConnected:   false,
        connected:        false,
        logger:           l,
        lock:             sync.Mutex{},
        sendLock:         sync.Mutex{},
        Subscriptions:    make(map[string]*BridgeClientSub),
        ConnectedChan:    make(chan bool),
        disconnectedChan: make(chan bool),
        inboundChan:      make(chan *frame.Frame)}
}

// Connect to broker endpoint.
func (ws *BridgeClient) Connect(url *url.URL, config *BrokerConnectorConfig) error {
    ws.lock.Lock()
    defer ws.lock.Unlock()
    if ws.logger != nil {
        ws.logger.Printf("connecting to fabric endpoint over %s", url.String())
    }

    c, _, err := websocket.DefaultDialer.Dial(url.String(), config.HttpHeader)
    if err != nil {
        return err
    }
    ws.WSc = c

    // handle incoming STOMP frames.
    go ws.handleIncomingSTOMPFrames()

    // go listen to the websocket
    go ws.listenSocket()

    stompHeaders := []string{
        frame.AcceptVersion,
        string(stomp.V12),
        frame.Login,
        config.Username,
        frame.Passcode,
        config.Password,
        frame.HeartBeat,
        fmt.Sprintf("%d,%d", config.HeartBeatOut.Milliseconds(), config.HeartBeatIn.Milliseconds())}
    for key, value := range config.STOMPHeader {
        stompHeaders = append(stompHeaders, key, value)
    }

    // send connect frame.
    ws.SendFrame(frame.New(frame.CONNECT, stompHeaders...))

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
    ws.SendFrame(subscribeFrame)
    return s
}

// send a payload to a destination
func (ws *BridgeClient) Send(destination, contentType string, payload []byte, opts ...func(fr *frame.Frame) error) {
    ws.lock.Lock()
    defer ws.lock.Unlock()

    // create send frame.
    sendFrame := frame.New(frame.SEND,
        frame.Destination, destination,
        frame.ContentLength, strconv.Itoa(len(payload)),
        frame.ContentType, contentType)

    // apply extra frame options such as adding extra headers
    for _, frameOpt := range opts {
        _ = frameOpt(sendFrame)
    }
    // add payload
    sendFrame.Body = payload

    // send frame
    go ws.SendFrame(sendFrame)

}

// send a STOMP frame down the WebSocket
func (ws *BridgeClient) SendFrame(f *frame.Frame) {
    ws.sendLock.Lock()
    defer ws.sendLock.Unlock()
    var b bytes.Buffer
    br := bufio.NewWriter(&b)
    sw := frame.NewWriter(br)

    // write frame to buffer
    sw.Write(f)
    w, _ := ws.WSc.NextWriter(websocket.TextMessage)
    defer w.Close()

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
            ws.inboundChan <- f
        }
    }
}

func (ws *BridgeClient) handleIncomingSTOMPFrames() {
    for {
        select {
        case <-ws.disconnectedChan:
            return
        case f := <-ws.inboundChan:
            switch f.Command {
            case frame.CONNECTED:
                if ws.logger != nil {
                    ws.logger.Printf("STOMP Client connected")
                }
                ws.stompConnected = true
                ws.connected = true
                ws.ConnectedChan <- true

            case frame.MESSAGE:
                for _, sub := range ws.Subscriptions {
                    if sub.Destination == f.Header.Get(frame.Destination) {
                        c := &model.MessageConfig{Payload: f.Body, Destination: sub.Destination}
                        sub.lock.RLock()
                        if sub.subscribed {
                            ws.sendResponseSafe(sub.C, model.GenerateResponse(c))
                        }
                        sub.lock.RUnlock()
                    }
                }

            case frame.ERROR:
                if ws.logger != nil {
                    ws.logger.Printf("STOMP ErrorDir received")
                }

                for _, sub := range ws.Subscriptions {
                    if sub.Destination == f.Header.Get(frame.Destination) {
                        c := &model.MessageConfig{Payload: f.Body, Err: errors.New("STOMP ErrorDir " + string(f.Body))}
                        sub.E <- model.GenerateError(c)
                    }
                }
            }
        }
    }
}

func (ws *BridgeClient) sendResponseSafe(C chan *model.Message, m *model.Message) {
    defer func() {
        if r := recover(); r != nil {
            if ws.logger != nil {
                ws.logger.Println("channel is closed, message undeliverable to closed channel.")
            }
        }
    }()
    C <- m
}

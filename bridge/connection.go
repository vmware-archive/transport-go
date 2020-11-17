// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
    "github.com/vmware/transport-go/model"
    "log"
    "sync"
)

type Connection interface {
    GetId() *uuid.UUID
    Subscribe(destination string) (Subscription, error)
    Disconnect() (err error)
    SendMessage(destination string, payload []byte) error
}

// Connection represents a Connection to a message broker.
type connection struct {
    id             *uuid.UUID
    useWs          bool
    conn           *stomp.Conn
    wsConn         *BridgeClient
    disconnectChan chan bool
    subscriptions  map[string]Subscription
    connLock       sync.Mutex
}

func (c *connection) GetId() *uuid.UUID{
    return c.id
}

// Subscribe to a destination, only one subscription can exist for a destination
func (c *connection) Subscribe(destination string) (Subscription, error) {
    // check if the subscription exists, if so, return it.
    if c == nil {
        return nil, fmt.Errorf("cannot subscribe to '%s', no connection to broker", destination)
    }

    c.connLock.Lock()
    if sub, ok := c.subscriptions[destination]; ok {
        c.connLock.Unlock()
        return sub, nil
    }
    c.connLock.Unlock()

    // use websocket, if not use stomp TCP.
    if c.useWs {
        return c.subscribeWs(destination)
    }

    return c.subscribeTCP(destination)
}

// Disconnect from broker, will close all channels
func (c *connection) Disconnect() (err error) {
    if c == nil {
        return fmt.Errorf("cannot disconnect, not connected")
    }
    if c.useWs {
        if c.wsConn != nil && c.wsConn.connected {
            defer c.cleanUpConnection()
            err = c.wsConn.Disconnect()
        }
    } else {
        if c.conn != nil {
            defer c.cleanUpConnection()
            err = c.conn.Disconnect()
        }
    }
    return err
}

func (c *connection) cleanUpConnection() {
    if c.conn != nil {
        c.conn = nil
    }
    if c.wsConn != nil {
        c.wsConn = nil
    }
}

func (c *connection) subscribeWs(destination string) (Subscription, error) {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c.wsConn != nil {
        wsSub := c.wsConn.Subscribe(destination)
        sub := &subscription{wsStompSub: wsSub, id: wsSub.Id, c: wsSub.C, destination: destination}
        c.subscriptions[destination] = sub
        return sub, nil
    }
    return nil, fmt.Errorf("cannot subscribe, websocket not connected / established")
}

func (c *connection) subscribeTCP(destination string) (Subscription, error) {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c.conn != nil {
        sub, _ := c.conn.Subscribe(destination, stomp.AckAuto)
        id := uuid.New()
        destChan := make(chan *model.Message)
        go c.listenTCPFrames(sub.C, destChan)
        bcSub := &subscription{stompTCPSub: sub, id: &id, c: destChan}
        c.subscriptions[destination] = bcSub
        return bcSub, nil
    }
    return nil, fmt.Errorf("no STOMP TCP connection established")
}

func (c *connection) listenTCPFrames(src chan *stomp.Message, dst chan *model.Message) {
    defer func() {
        if r := recover(); r != nil {
            log.Println("subscription is closed, message undeliverable to closed channel.")
        }
    }()
    for {
        f := <-src
        var body []byte
        var dest string
        if f != nil && f.Body != nil {
            body = f.Body
        }
        if f!= nil && len(f.Destination) > 0  {
            dest = f.Destination
        }
        if f != nil {
            cf := &model.MessageConfig{Payload: body, Destination: dest}
            m := model.GenerateResponse(cf)
            dst <- m
        }
    }
}

// Send a []byte payload to a destination.
func (c *connection) SendMessage(destination string, payload []byte) error {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c != nil && !c.useWs && c.conn != nil {
        c.conn.Send(destination, "application/json", payload, nil)
        return nil
    }
    if c != nil && c.useWs && c.wsConn != nil {
        c.wsConn.Send(destination, payload)
        return nil
    }
    return fmt.Errorf("cannot send message, no connection")

}

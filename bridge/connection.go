package bridge

import (
    "bifrost/bus"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
    "sync"
)

// abstraction for connection types.
type Connection struct {
    useWs          bool
    conn           *stomp.Conn
    wsConn         *BridgeClient
    disconnectChan chan bool
    subscriptions  map[string]*Subscription
    connLock       sync.Mutex
}

func (c *Connection) Subscribe(destination string) (*Subscription, error) {
    // check if the subscription exists, if so, return it.
    if c == nil {
        return nil, fmt.Errorf("cannot subscribe to '%s', no connection to broker", destination)
    }

    if sub, ok := c.subscriptions[destination]; ok {
        return sub, nil
    }

    // use websocket, if not use stomp TCP.
    if c.useWs {
        return c.subscribeWs(destination)
    }

    return c.subscribeTCP(destination)
}

func (c *Connection) Disconnect() (err error) {
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

func (c *Connection) cleanUpConnection() {
    if c.conn != nil {
        c.conn = nil
    }
    if c.wsConn != nil {
        c.wsConn = nil
    }
}

func (c *Connection) subscribeWs(destination string) (*Subscription, error) {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c.wsConn != nil {
        wsSub := c.wsConn.Subscribe(destination)
        sub := &Subscription{wsStompSub: wsSub, Id: wsSub.Id, C: wsSub.C, Destination: destination}
        c.subscriptions[destination] = sub
        return sub, nil
    }
    return nil, fmt.Errorf("cannot subscribe, websocket not connected / established")
}

func (c *Connection) subscribeTCP(destination string) (*Subscription, error) {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c.conn != nil {
        sub, _ := c.conn.Subscribe(destination, stomp.AckAuto)
        id := uuid.New()
        destChan := make(chan *bus.Message)
        go c.listenTCPFrames(sub.C, destChan)
        bcSub := &Subscription{stompTCPSub: sub, Id: &id, C: destChan}
        c.subscriptions[destination] = bcSub
        return bcSub, nil
    }
    return nil, fmt.Errorf("no STOMP TCP conncetion established")
}

func (c *Connection) listenTCPFrames(src chan *stomp.Message, dst chan *bus.Message) {
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
            cf := &bus.MessageConfig{Payload: body, Destination: dest}
            m := bus.GenerateResponse(cf)
            dst <- m
        }
    }
}

func (c *Connection) SendMessage(destination string, payload []byte) error {
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

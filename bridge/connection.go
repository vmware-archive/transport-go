package bridge

import (
    "bifrost/bus"
    "encoding/json"
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
    if sub, ok := c.subscriptions[destination]; ok {
        return sub, nil
    }

    // use websocket, if not use stomp TCP.
    if c.useWs {
        return c.subscribeWs(destination)
    }

    return c.subscribeTCP(destination)
}

func (c *Connection) Disconnect() error {

    if c.useWs {
        if c.wsConn != nil && c.wsConn.connected {
            //c.disconnectChan <- true
            return c.wsConn.Disconnect()
        }
    }
    return nil
}

func (c *Connection) subscribeWs(destination string) (*Subscription, error) {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c.wsConn != nil {
        wsSub := c.wsConn.Subscribe(destination)
        sub := &Subscription{wsStompSub: wsSub, Id: wsSub.Id, C: wsSub.C, Destination: destination}
        return sub, nil
    }
    return nil, fmt.Errorf("cannot subscribe, websocket not connected / established")
}

func (c *Connection) subscribeTCP(destination string) (*Subscription, error) {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if c.conn != nil {
        sub, err := c.conn.Subscribe(destination, stomp.AckAuto)
        if err != nil {
            return nil, err
        }
        id := uuid.New()
        destChan := make(chan *bus.Message)
        go c.listenTCPFrames(sub.C, destChan)
        bcSub := &Subscription{stompTCPSub: sub, Id: &id, C: destChan}
        return bcSub, nil
    }
    return nil, fmt.Errorf("no STOMP TCP conncetion established")
}

func (c *Connection) listenTCPFrames(src chan *stomp.Message, dst chan *bus.Message) {
    for {
        f := <-src
        cf := &bus.MessageConfig{Payload: f.Body, Destination: f.Destination}
        m := bus.GenerateResponse(cf)
        dst <- m
    }
}

func (c *Connection) SendMessage(destination string, msg *bus.Message) error {
    c.connLock.Lock()
    defer c.connLock.Unlock()
    if pl, err := json.Marshal(msg.Payload); err != nil {
        return err
    } else {
        if !c.useWs {
            if err := c.conn.Send(destination, "application/json", pl, nil); err != nil {
                return err
            }
        } else {
            c.wsConn.Send(destination, pl)
        }
    }
    return nil

}

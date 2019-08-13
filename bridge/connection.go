package bridge

import (
    "bifrost/bus"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
)

// abstraction for connection types.
type Connection struct {
    useWs          bool
    conn           *stomp.Conn
    wsConn         *BridgeClient
    disconnectChan chan bool
    subscriptions  map[string]*Subscription
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
            c.disconnectChan <- true
            return c.wsConn.Disconnect()
        }
    }
    return nil
}


func (c *Connection) subscribeWs(destination string) (*Subscription, error) {
    if c.wsConn != nil {
        wsSub := c.wsConn.Subscribe(destination)
        sub := &Subscription{wsStompSub: wsSub, Id: wsSub.Id, C: wsSub.C}
        return sub, nil
    }
    return nil, fmt.Errorf("cannot subscribe, websocket not connected / established")
}

func (c *Connection) subscribeTCP(destination string) (*Subscription, error) {
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

//
//func (c *Connection) Unsubscribe(destination string) error {
//    // check if the subscription exists, if not, fail.
//    if c.subscriptions[destination] != nil {
//        return fmt.Errorf("unable to unsubscribe, no subscription for destination %s", destination)
//    }
//    sub := c.subscriptions[destination]
//
//    if c.useWs && sub.wsStompSub != nil {
//        sub.wsStompSub.Unsubscribe()
//        return nil
//    }
//
//    if !c.useWs && sub.stompTCPSub != nil {
//        return sub.stompTCPSub.Unsubscribe()
//    }
//
//    return fmt.Errorf("cannot unsubscribe from destination %s, no connection", destination)
//}


func (c *Connection) listenTCPFrames(src chan *stomp.Message, dst chan *bus.Message) {
    for {
        f := <-src
        cf := &bus.MessageConfig{Payload: f.Body, Destination: f.Destination}
        m := bus.GenerateResponse(cf)
        dst <- m
        <-c.disconnectChan
        return
    }

}

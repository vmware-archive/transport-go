package bridge

import (
    "bifrost/bus"
    "encoding/json"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
    "log"
    "net/url"
)

type BrokerConnector interface {
    Connect(config *BrokerConnectorConfig) (*Connection, error)
    ConnectWs(config *BrokerConnectorConfig) (*Connection, error)
    Subscribe(destination string) (*Subscription, error)
    Unsubscribe(destination string) error
    Disconnect() error
    SendMessage(destination string, message *bus.Message) error
}

type brokerConnector struct {
    c             *Connection
    connected     bool
    subscriptions map[string]*Subscription
    bus           bus.EventBus
}

func NewBrokerConnector() BrokerConnector {
    return &brokerConnector{connected: false, subscriptions: make(map[string]*Subscription), bus: bus.GetBus()}
}

func checkConfig(config *BrokerConnectorConfig) error {
    if config.ServerAddr == "" {
        return fmt.Errorf("config invalid, config missing server address")
    }
    if config.Username == "" {
        return fmt.Errorf("config invalid, config missing username")
    }
    if config.Password == "" {
        return fmt.Errorf("config invalid, config missing password")
    }
    return nil
}

func (bc *brokerConnector) Connect(config *BrokerConnectorConfig) (*Connection, error) {

    err := checkConfig(config)
    if err != nil {
        return nil, err
    }

    if config.HostHeader == "" {
        config.HostHeader = "/"
    }

    var options = []func(*stomp.Conn) error{
        stomp.ConnOpt.Login(config.Username, config.Password),
        stomp.ConnOpt.Host(config.HostHeader),
    }

    conn, err := stomp.Dial("tcp", config.ServerAddr, options...)
    if err != nil {
        return nil, err
    }
    bcConn := &Connection{Conn: conn}
    bc.c = bcConn
    bc.connected = true
    return bcConn, nil
}

func (bc *brokerConnector) ConnectWs(config *BrokerConnectorConfig) (*Connection, error) {

    err := checkConfig(config)
    if err != nil {
        return nil, err
    }


    // TODO: Make sure 'ws' is moved to config.
    u := url.URL{Scheme: "ws", Host: config.ServerAddr, Path: config.WSPath}
    c := NewBridgeWsClient()
    err = c.Connect(&u, nil)
    if err != nil {
        log.Panicf("cannot connect to host %s via path %s, stopping", config.ServerAddr, config.WSPath)
    }

    bcConn := &Connection{WsConn: c}
    bc.c = bcConn
    bc.connected = true
    return bcConn, nil
}


func (bc *brokerConnector) Subscribe(destination string) (*Subscription, error) {

    // check if the subscription exists, if so, return it.
    if sub, ok := bc.subscriptions[destination]; ok {
        return sub, nil
    }

    sub, err := bc.c.Conn.Subscribe(destination, stomp.AckAuto)
    if err != nil {
        return nil, err
    }
    id := uuid.New()
    bcSub := &Subscription{StompSub: sub, Id: &id}
    return bcSub, nil
}

func (bc *brokerConnector) Unsubscribe(destination string) error {
    // check if the subscription exists, if not, fail.
    if bc.subscriptions[destination] != nil {
        return fmt.Errorf("unable to unsubscribe, no subscription for destination %s", destination)
    }
    sub := bc.subscriptions[destination]

    if sub.StompSub.Active() {
        return sub.StompSub.Unsubscribe()
    }
    return nil
}

func (bc *brokerConnector) Disconnect() error {
    bc.connected = false
    return bc.c.Conn.Disconnect()
}

func (bc *brokerConnector) SendMessage(destination string, msg *bus.Message) error {
    if bc.connected {

        if pl, err := json.Marshal(msg.Payload); err != nil {
            return err
        } else {
            if err := bc.c.Conn.Send(destination, "text/plain", pl, nil); err != nil {
                return err
            }
        }
        return nil

    } else {
        return fmt.Errorf("unable to send message, not connected to broker")
    }
}

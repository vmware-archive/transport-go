package bridge

import (
    "bifrost/bus"
    "fmt"
    "github.com/go-stomp/stomp"
    "log"
    "net/url"
    "sync"
)

type BrokerConnector interface {
    Connect(config *BrokerConnectorConfig) (*Connection, error)
    //Subscribe(destination string) (*Subscription, error)
    //Unsubscribe(destination string) error
    //Disconnect() error
    //SendMessage(destination string, message *bus.Message) error
}

type brokerConnector struct {
    c             *Connection
    config        *BrokerConnectorConfig
    connected     bool
    bus           bus.EventBus
}

func NewBrokerConnector() BrokerConnector {
    return &brokerConnector{connected: false, bus: bus.GetBus()}
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

    // use different mechanism for WS connections.
    if config.UseWS {
        return bc.connectWs(config)
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
    bcConn := &Connection{
        conn: conn,
        subscriptions: make(map[string]*Subscription),
        useWs: false,
        connLock: sync.Mutex{},
        disconnectChan: make(chan bool)}
    bc.c = bcConn
    bc.connected = true
    bc.config = config
    return bcConn, nil
}

func (bc *brokerConnector) connectWs(config *BrokerConnectorConfig) (*Connection, error) {

    // TODO: Make sure 'ws' is moved to config.
    u := url.URL{Scheme: "ws", Host: config.ServerAddr, Path: config.WSPath}
    c := NewBridgeWsClient()
    err := c.Connect(&u, nil)
    if err != nil {
        log.Panicf("cannot connect to host %s via path %s, stopping", config.ServerAddr, config.WSPath)
    }

    bcConn := &Connection{
        wsConn: c,
        subscriptions: make(map[string]*Subscription),
        useWs: true,
        connLock: sync.Mutex{},
        disconnectChan: make(chan bool)}
    bc.c = bcConn
    bc.connected = true
    return bcConn, nil
}

//func (bc *brokerConnector) Disconnect() error {
//    bc.connected = false
//    return bc.c.conn.Disconnect()
//}

//func (bc *brokerConnector) SendMessage(destination string, msg *bus.Message) error {
//    if bc.connected {
//
//        if pl, err := json.Marshal(msg.Payload); err != nil {
//            return err
//        } else {
//            if err := bc.c.conn.Send(destination, "text/plain", pl, nil); err != nil {
//                return err
//            }
//        }
//        return nil
//
//    } else {
//        return fmt.Errorf("unable to send message, not connected to broker")
//    }
//}

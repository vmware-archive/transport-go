package bridge

import (
    "bufio"
    "bytes"
    "fmt"
    "github.com/go-stomp/stomp/frame"
    "github.com/go-stomp/stomp/server"
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    "log"
    "net"
    "net/http"
    "net/http/httptest"
    "net/url"
    "testing"
)

var upgrader = websocket.Upgrader{}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer c.Close()
    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            break
        }

        br := bytes.NewReader(message)
        sr := frame.NewReader(br)
        f, _ := sr.Read()

        var sendFrame *frame.Frame

        switch f.Command {
        case frame.CONNECT:
            sendFrame = frame.New(frame.CONNECTED,
                frame.ContentType, "application/json")

        }
        var bb bytes.Buffer
        bw := bufio.NewWriter(&bb)
        sw := frame.NewWriter(bw)
        sw.Write(sendFrame)

        err = c.WriteMessage(mt, bb.Bytes())
        if err != nil {
            break
        }
    }
}

//var srv Server
var testBrokerAddress = ":8992"
var httpServer *httptest.Server
var tcpServer net.Listener
var webSocketURLChan = make(chan string)

func runStompBroker() {
    l, err := net.Listen("tcp", testBrokerAddress)
    if err != nil {
        log.Fatalf("failed to listen: %s", err.Error())
    }
    defer func() { l.Close() }()

    log.Println("TCP listening on", l.Addr().Network(), l.Addr().String())
    server.Serve(l)
    tcpServer = l
}

func runWebSocketEndPoint() {
    s := httptest.NewServer(http.HandlerFunc(websocketHandler))
    log.Println("WebSocket listening on", s.Listener.Addr().Network(), s.Listener.Addr().String())
    httpServer = s
    webSocketURLChan <- s.URL
}

func TestBrokerConnector_BadConfig(t *testing.T) {
    tt := []struct {
        test   string
        config *BrokerConnectorConfig
        err    error
    }{
        {
            "Missing address from config",
            &BrokerConnectorConfig{Username: "guest", Password: "guest"},
            fmt.Errorf("config invalid, config missing server address")},
        {
            "Missing username from config",
            &BrokerConnectorConfig{ServerAddr: "somewhere:000"},
            fmt.Errorf("config invalid, config missing username")},
        {
            "Missing password from config",
            &BrokerConnectorConfig{Username: "hi", ServerAddr: "somewhere:000"},
            fmt.Errorf("config invalid, config missing password")},
    }

    for _, tc := range tt {
        t.Run(tc.test, func(t *testing.T) {
            bc := NewBrokerConnector()
            c, err := bc.Connect(tc.config)
            assert.Nil(t, c)
            assert.NotNil(t, err)
            assert.Equal(t, tc.err, err)
        })
    }
}

func TestBrokerConnector_ConnectBroker(t *testing.T) {
    go runStompBroker()
    go runWebSocketEndPoint()
    u := <- webSocketURLChan
    url, _ := url.Parse(u)
    host, port, _ := net.SplitHostPort(url.Host)
    testHost := host + ":" + port

    tt := []struct {
        test   string
        config *BrokerConnectorConfig
    }{
        {
            "Connect via websocket",
            &BrokerConnectorConfig{
                Username: "guest", Password: "guest", UseWS:true, WSPath:"/", ServerAddr: testHost}},
        {
            "Connect via TCP",
            &BrokerConnectorConfig{
                Username: "guest", Password: "guest", ServerAddr: testBrokerAddress}},
    }

    for _, tc := range tt {
        t.Run(tc.test, func(t *testing.T) {
            bc := NewBrokerConnector()
            c, err := bc.Connect(tc.config)
            assert.NotNil(t, c)
            assert.Nil(t, err)
            if tc.config.UseWS {
              assert.NotNil(t, c.wsConn)
            }
            if !tc.config.UseWS {
               assert.NotNil(t, c.conn)
            }
        })
    }
}

func TestBrokerConnector_ConnectBrokerFail(t *testing.T) {
    tt := []struct {
        test   string
        config *BrokerConnectorConfig
    }{
        {
            "Connect via websocket fails with bad address",
            &BrokerConnectorConfig{
                Username: "guest", Password: "guest", UseWS:true, WSPath:"/", ServerAddr: "nowhere"}},
        {
            "Connect via TCP fails with bad address",
            &BrokerConnectorConfig{
                Username: "guest", Password: "guest", ServerAddr: "somewhere"}},
    }

    for _, tc := range tt {
        t.Run(tc.test, func(t *testing.T) {
            bc := NewBrokerConnector()
            c, err := bc.Connect(tc.config)
            assert.Nil(t, c)
            assert.NotNil(t, err)
        })
    }
}

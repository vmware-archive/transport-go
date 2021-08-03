// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
    "crypto/tls"
    "fmt"
    "github.com/go-stomp/stomp/frame"
    "github.com/go-stomp/stomp/server"
    "github.com/stretchr/testify/assert"
    "log"
    "net"
    "net/http"
    "net/http/httptest"
    "net/url"
    "testing"
)

var webSocketURLChanTLS = make(chan string)
var websocketURLTLS string

//var srv Server
var testTLS = &tls.Config{
    InsecureSkipVerify: true,
    MinVersion:         tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
        tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_RSA_WITH_AES_256_CBC_SHA,
    },
}

func runWebSocketEndPointTLS() {
    s := httptest.NewUnstartedServer(http.HandlerFunc(websocketHandler))
    s.TLS = testTLS
    s.StartTLS()
    log.Println("WebSocket listening on", s.Listener.Addr().Network(), s.Listener.Addr().String(), "(TLS)")
    httpServer = s
    webSocketURLChanTLS <- s.URL
}

func runStompBrokerTLS() {
    l, err := net.Listen("tcp", ":51582")
    if err != nil {
        log.Fatalf("failed to listen: %s", err.Error())
    }
    defer func() { l.Close() }()

    log.Println("TCP listening on", l.Addr().Network(), l.Addr().String(), "(TLS)")
    server.Serve(l)
    tcpServer = l
}

func init() {
    go runStompBrokerTLS()
    go runWebSocketEndPointTLS()

    websocketURLTLS = <-webSocketURLChanTLS
}

func TestBrokerConnector_ConnectBroker_Invalid_TLS_Cert(t *testing.T) {
    url, _ := url.Parse(websocketURLTLS)
    host, port, _ := net.SplitHostPort(url.Host)
    testHost := host + ":" + port

    brokerConfig := &BrokerConnectorConfig{
        Username: "guest",
        Password: "guest",
        UseWS: true,
        WebSocketConfig: &WebSocketConfig{
            WSPath:    "/fabric",
            UseTLS:    true,
            TLSConfig: testTLS,
            CertFile:  "nothing",
            KeyFile:   "nothing",
        },
        ServerAddr: testHost,
    }
    bc := NewBrokerConnector()
    _, err := bc.Connect(brokerConfig, true)

    assert.NotNil(t, err)
}

func TestBrokerConnector_ConnectBroker_TLS(t *testing.T) {
    url, _ := url.Parse(websocketURLTLS)
    host, port, _ := net.SplitHostPort(url.Host)
    testHost := host + ":" + port

    tt := []struct {
        test   string
        config *BrokerConnectorConfig
    }{
        {
            "Connect via websocket with TLS",
            &BrokerConnectorConfig{
                Username: "guest",
                Password: "guest",
                WebSocketConfig: &WebSocketConfig{
                    WSPath: "/",
                    UseTLS: true,
                    CertFile: "test_server.crt",
                    KeyFile: "test_server.key",
                    TLSConfig: testTLS,
                },
                UseWS: true,
                STOMPHeader: map[string]string{
                    "access-token": "test",
                },
                ServerAddr: testHost},
        },
    }

    for _, tc := range tt {
        t.Run(tc.test, func(t *testing.T) {

            // connect
            bc := NewBrokerConnector()
            c, err := bc.Connect(tc.config, true)

            if err != nil {
                fmt.Printf("unable to connect, error: %e", err)
            }

            assert.NotNil(t, c)
            assert.Nil(t, err)
            if tc.config.UseWS {
                assert.NotNil(t, c.(*connection).wsConn)
            }
            if !tc.config.UseWS {
                assert.NotNil(t, c.(*connection).conn)
            }

            m, _ := c.Subscribe("/topic/test-topic")
            go func() {
                err = c.SendJSONMessage("/topic/test-topic", []byte("{}"), func(frame *frame.Frame) error {
                    frame.Header.Set("access-token", "test")
                    return nil
                })
                assert.Nil(t, err)
            }()
            msg := <-m.GetMsgChannel()
            b := msg.Payload.([]byte)
            assert.EqualValues(t, "happy baby melody!", string(b))


            // disconnect
            err = c.Disconnect()
            assert.Nil(t, err)
            if tc.config.UseWS {
                assert.Nil(t, c.(*connection).wsConn)
            }
            if !tc.config.UseWS {
                assert.Nil(t, c.(*connection).conn)
            }
        })
    }
}

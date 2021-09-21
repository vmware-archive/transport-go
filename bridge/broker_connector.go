// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
	"crypto/tls"
	"fmt"
	"github.com/go-stomp/stomp/v3"
	"github.com/google/uuid"
	"net/url"
	"sync"
)

// BrokerConnector is used to connect to a message broker over TCP or WebSocket.
type BrokerConnector interface {
	Connect(config *BrokerConnectorConfig, enableLogging bool) (Connection, error)
}

type brokerConnector struct {
	c         Connection
	config    *BrokerConnectorConfig
	connected bool
}

// Create a new broker connector
func NewBrokerConnector() BrokerConnector {
	return &brokerConnector{connected: false}
}

func checkConfig(config *BrokerConnectorConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	if config.ServerAddr == "" {
		return fmt.Errorf("config invalid, config missing server address")
	}
	if config.Username == "" {
		return fmt.Errorf("config invalid, config missing username")
	}
	if config.Password == "" {
		return fmt.Errorf("config invalid, config missing password")
	}

	// if TLS is being used and no default values are passed, use defaults, so we don't add
	// cognitive load to using the client with just the basics.
	if config.WebSocketConfig != nil && config.WebSocketConfig.UseTLS && config.WebSocketConfig.TLSConfig == nil {
		var basicTLSConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		config.WebSocketConfig.TLSConfig = basicTLSConfig
	}
	return nil
}

// Connect to broker using supplied connector config.
func (bc *brokerConnector) Connect(config *BrokerConnectorConfig, enableLogging bool) (Connection, error) {

	err := checkConfig(config)
	if err != nil {
		return nil, err
	}

	// use different mechanism for WS connections.
	if config.UseWS {
		return bc.connectWs(config, enableLogging)
	}

	return bc.connectTCP(config, err)
}

func (bc *brokerConnector) connectTCP(config *BrokerConnectorConfig, err error) (Connection, error) {
	if config.HostHeader == "" {
		config.HostHeader = "/"
	}
	var options = []func(*stomp.Conn) error{
		stomp.ConnOpt.Login(config.Username, config.Password),
		stomp.ConnOpt.Host(config.HostHeader),
		stomp.ConnOpt.HeartBeat(config.HeartBeatOut, config.HeartBeatIn),
	}

	if config.STOMPHeader != nil {
		for key, value := range config.STOMPHeader {
			options = append(options, stomp.ConnOpt.Header(key, value))
		}
	}

	conn, err := stomp.Dial("tcp", config.ServerAddr, options...)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	bcConn := &connection{
		id:             &id,
		conn:           conn,
		subscriptions:  make(map[string]Subscription),
		useWs:          false,
		connLock:       sync.Mutex{},
		disconnectChan: make(chan bool)}
	bc.c = bcConn
	bc.connected = true
	bc.config = config
	return bcConn, nil
}

func (bc *brokerConnector) connectWs(config *BrokerConnectorConfig, enableLogging bool) (Connection, error) {
	wsScheme := "ws"
	if config.WebSocketConfig.UseTLS {
		wsScheme += "s"
	}

	u := url.URL{Scheme: wsScheme, Host: config.ServerAddr, Path: config.WebSocketConfig.WSPath}
	c := NewBridgeWsClient(enableLogging)
	err := c.Connect(&u, config)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to host '%s' via path '%s', stopping", config.ServerAddr, config.WebSocketConfig.WSPath)
	}
	id := uuid.New()
	bcConn := &connection{
		id:             &id,
		wsConn:         c,
		subscriptions:  make(map[string]Subscription),
		useWs:          true,
		connLock:       sync.Mutex{},
		disconnectChan: make(chan bool)}
	bc.c = bcConn
	bc.connected = true
	return bcConn, nil
}

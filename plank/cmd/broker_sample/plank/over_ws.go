// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package plank

import (
	"crypto/tls"
	"encoding/json"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/plank/utils"
	"net/http"
	"os"
	"time"
)

// ListenViaWS listens to sample channel services.PingPongServiceChan over WebSocket. before running this
// function make sure you have a Plank instance running in default port (30080). also if you run it with
// TLS enabled, make sure that your TLS certificate and private key are where the demo app expects them to be
// (cert/fullchain.pem and cert/server.key) and that these match those used in Plank.
func ListenViaWS(c chan os.Signal, useTLS bool) {
	var err error
	b := bus.GetBus()
	cm := b.GetChannelManager()

	// create a local service channel. this is the channel we'll be mapping to the
	// galactic channel from the remote Plank instance.
	cm.CreateChannel(services.PingPongServiceChan)

	// connect to the broker
	broker, err := b.ConnectBroker(getBrokerConnectorConfig(useTLS))
	if err != nil {
		utils.Log.Fatalln("conn error", err)
	}

	// mark channel galactic by linking it to the broker we have just connected to
	if err = b.GetChannelManager().MarkChannelAsGalactic(services.PingPongServiceChan, "/queue/"+services.PingPongServiceChan, broker); err != nil {
		utils.Log.Fatalln(err)
	}

	// listen to messages arriving in the channel and print out messages by setting up a handler
	hd, err := b.ListenStream(services.PingPongServiceChan)
	if err != nil {
		utils.Log.Fatalln(err)
	}
	hd.Handle(func(message *model.Message) {
		response := &model.Response{}
		err := json.Unmarshal(message.Payload.([]byte), response)
		if err != nil {
			utils.Log.Fatalln(err)
		}
		utils.Log.Printf("%v", response)
	}, func(err error) {
		utils.Log.Fatalln(err)
	})
	utils.Log.Infoln("waiting for messages")

	// now that we are listening to the channel send a request to the ping pong service to receive a message back.
	time.Sleep(2 * time.Second)
	md := &model.Request{Request: "ping-get", Payload: "hello"}
	m, _ := json.Marshal(md)
	err = broker.SendJSONMessage("/pub/queue/"+services.PingPongServiceChan, m)
	if err != nil {
		utils.Log.Fatalln(err)
	}

	<-c
}

// getBrokerConnectorConfig returns a basic *bridge.BrokerConnectorConfig based on the
func getBrokerConnectorConfig(useTLS bool) *bridge.BrokerConnectorConfig {
	config := &bridge.BrokerConnectorConfig{
		Username:   "guest",
		Password:   "guest",
		ServerAddr: "localhost:30080",
		UseWS:      true,
		WebSocketConfig: &bridge.WebSocketConfig{
			WSPath: "/ws",
			UseTLS: useTLS,
		},
		HeartBeatOut: 30 * time.Second,
		STOMPHeader:  map[string]string{},
		HttpHeader: http.Header{
			"Sec-Websocket-Protocol": {"v12.stomp"},
		},
	}

	if useTLS {
		config.WebSocketConfig.CertFile = "cert/fullchain.pem"
		config.WebSocketConfig.KeyFile = "cert/server.key"
		config.WebSocketConfig.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
	}
	return config
}

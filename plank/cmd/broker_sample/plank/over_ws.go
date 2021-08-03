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

func ListenViaWS(c chan os.Signal, useTLS bool) {
	var err error
	b := bus.GetBus()
	cm := b.GetChannelManager()
	cm.CreateChannel(services.PingPongServiceChan)

	broker, err := b.ConnectBroker(getBrokerConnectorConfig(useTLS))
	if err != nil {
		utils.Log.Fatalln("conn error", err)
	}

	if err = b.GetChannelManager().MarkChannelAsGalactic(services.PingPongServiceChan, "/queue/"+services.PingPongServiceChan, broker); err != nil {
		utils.Log.Fatalln(err)
	}

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

	time.Sleep(2 * time.Second)
	md := &model.Request{Request: "ping2", Payload: "hello"}
	m, _ := json.Marshal(md)
	err = broker.SendJSONMessage("/pub/queue/"+services.PingPongServiceChan, m)
	if err != nil {
		utils.Log.Fatalln(err)
	}

	<-c
}

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
		STOMPHeader: map[string]string{
			"access-token": "something",
			"hooo":         "ddffd",
		},
		HttpHeader: http.Header{
			"Sec-Websocket-Protocol": {"v12.stomp, access-token.something"},
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

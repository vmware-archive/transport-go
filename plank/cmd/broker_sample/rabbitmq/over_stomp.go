// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package rabbitmq

import (
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/plank/utils"
	"os"
	"time"
)

// ListenViaStomp directly connects to a RabbitMQ instance via TCP port 61613 that RabbitMQ STOMP plugin
// exposes, and listens on topic "something.somewhere". also it sends out a test message to the topic
// after a one second delay. note that RabbitMQ STOMP plugin routes messages arriving from the client
// to the amq.topic exchange. for details see producer.go
func ListenViaStomp(c2 chan os.Signal) {
	b := bus.GetBus()
	bus.EnableLogging(true)

	// connect to the RabbitMQ STOMP endpoint
	broker, err := b.ConnectBroker(&bridge.BrokerConnectorConfig{
		Username:     "guest",
		Password:     "guest",
		ServerAddr:   "localhost:61613",
		HeartBeatOut: 30 * time.Second,
		STOMPHeader: map[string]string{
			"access-token": "something",
		},
	})
	if err != nil {
		utils.Log.Fatalln("conn error", err)
	}

	// send a message to topic named "something.somewhere" after one second
	go func() {
		time.Sleep(1 * time.Second)
		broker.SendMessage("/topic/something.somewhere", "text/plain", []byte("i can send too!"))
	}()

	// subscribe to topic named "something.somewhere"
	subs, err := broker.Subscribe("/topic/something.somewhere")
	if err != nil {
		utils.Log.Fatalln(err)
	}

	// get the message channel from the subscription and print out messages as soon as they arrive on the channel
	c := subs.GetMsgChannel()
	go func() {
		for msg := range c {
			utils.Log.Infoln(msg)
		}
	}()

	utils.Log.Infoln("waiting for messages")
	<-c2
}

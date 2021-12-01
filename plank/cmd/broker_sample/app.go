// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"github.com/streadway/amqp"
	"github.com/vmware/transport-go/plank/cmd/broker_sample/plank"
	"github.com/vmware/transport-go/plank/cmd/broker_sample/rabbitmq"
	"github.com/vmware/transport-go/plank/utils"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main app to test drive transport's broker APIs. provide LISTEN_METHOD and PRODUCE_MESSAGE_ON_RABBITMQ as
// environment variables to control the behavior of this app.
// LISTEN_METHOD: decides how/where the demo app should receive messages from (plank_ws, plank_stomp, rbmq_amqp, rbmq_stomp)
// PRODUCE_MESSAGE_ON_RABBITMQ: when set to 1 it sends a dummy message every two seconds
// WS_USE_TLS: when set to 1, connection to Plank WebSocket will be made through wss protocol instead of ws
func main() {
	listenMethod := os.Getenv("LISTEN_METHOD")
	produceMessage := os.Getenv("PRODUCE_MESSAGE_ON_RABBITMQ")
	wsUseTls := os.Getenv("WS_USE_TLS") == "1"

	var ch *amqp.Channel
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL)

	// if PRODUCE_MESSAGE_ON_RABBITMQ is set to 1 then randomly send a message to a sample topic every two seconds
	if produceMessage == "1" {
		go func() {
			conn, err := rabbitmq.GetNewConnection("amqp://guest:guest@localhost:5672")
			if err != nil {
				utils.Log.Fatalln(err)
			}
			if ch, err = rabbitmq.GetNewChannel(conn); err != nil {
				utils.Log.Fatalln(err)
			}
			defer ch.Close()

			// send a message every two seconds
			for {
				time.Sleep(2 * time.Second)
				if err = rabbitmq.SendTopic(ch); err != nil {
					utils.Log.Errorln(err)
					c <- syscall.SIGKILL
				}
			}
		}()
	}

	switch listenMethod {
	case "plank_ws":
		// listen to the sample channel on Plank through WebSocket.
		// need to have a Plank instance running at default port (30080)
		plank.ListenViaWS(c, wsUseTls)
		break
	case "plank_stomp":
		// listen to the sample channel on Plank through STOMP exposed at TCP port 61613
		plank.ListenViaStomp(c)
	case "rbmq_amqp":
		// listen to the sample channel through AMQP
		// need to have a RabbitMQ instance running at default port
		rabbitmq.ListenViaAmqp(c)
		break
	case "rbmq_stomp":
		// listen to the sample channel through STOMP
		// need to have a RabbitMQ instance with STOMP plugin enabled (which listens at port 61613)
		rabbitmq.ListenViaStomp(c)
	}
}

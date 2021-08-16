// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package rabbitmq

import (
	"github.com/vmware/transport-go/plank/utils"
	"os"
)

// ListenViaAmqp directly connects to a RabbitMQ instance, sets up a sample topic exchange and
// creates a queue that is bound with routing key "something.somewhere". this method is not using
// any of Transport function but is useful to observe messages that originated from Transport.
// for more details about RabbitMQ Golang examples refer to https://www.rabbitmq.com/getstarted.html
func ListenViaAmqp(c2 chan os.Signal) {
	// open a new RabbitMQ connection
	conn, err := GetNewConnection("amqp://guest:guest@localhost:5672")
	if err != nil {
		utils.Log.Fatalln(err)
	}

	// acquire a new channel
	ch, err := conn.Channel()
	if err != nil {
		utils.Log.Fatalln(err)
	}
	defer ch.Close()

	// declare a new topic exchange
	if err = ch.ExchangeDeclare(
		"logs_topic",
		"topic",
		true,
		false,
		false,
		false,
		nil); err != nil {
		utils.Log.Fatalln(err)
	}

	// declare a new queue
	q, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil)
	if err != nil {
		utils.Log.Fatalln(err)
	}

	// bind the queue with the exchange
	if err = ch.QueueBind(
		q.Name,
		"something.somewhere",
		"logs_topic",
		false,
		nil); err != nil {
		utils.Log.Fatalln(err)
	}

	// consume the queue and acquire a channel for incoming messages
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		true,
		false,
		false,
		nil)
	if err != nil {
		utils.Log.Fatalln(err)
	}

	// print out messages as soon as they arrive
	go func() {
		for m := range msgs {
			utils.Log.Info(m)
		}
	}()

	utils.Log.Infoln("waiting for messages")
	<-c2
}

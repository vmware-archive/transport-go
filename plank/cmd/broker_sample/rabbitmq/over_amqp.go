// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package rabbitmq

import (
	"github.com/vmware/transport-go/plank/utils"
	"os"
)

func ListenViaAmqp(c2 chan os.Signal) {
	conn, err := GetNewConnection("amqp://guest:guest@localhost:5672")
	if err != nil {
		utils.Log.Fatalln(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		utils.Log.Fatalln(err)
	}
	defer ch.Close()

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

	if err = ch.QueueBind(
		q.Name,
		"something.somewhere",
		"logs_topic",
		false,
		nil); err != nil {
		utils.Log.Fatalln(err)
	}

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

	go func() {
		for m := range msgs {
			utils.Log.Info(m)
		}
	}()

	utils.Log.Infoln("waiting for messages")
	<-c2
}

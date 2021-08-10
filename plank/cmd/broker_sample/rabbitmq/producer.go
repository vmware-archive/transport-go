// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package rabbitmq

import (
	"github.com/streadway/amqp"
	"os"
)

// GetNewConnection returns a new *amqp.Connection and error if any
func GetNewConnection(url string) (conn *amqp.Connection, err error) {
	conn, err = amqp.Dial(url)
	return
}

// GetNewChannel returns a new *amqp.Channel from the passed *amqp.Connection instance and error if any
func GetNewChannel(conn *amqp.Connection) (ch *amqp.Channel, err error) {
	ch, err = conn.Channel()
	return
}

// SendTopic sends a message to a topic exchange with routing key "something.somewhere".
// if environment variable LISTEN_METHOD is set to rbmq_stomp then the excahgne is set to
// amq.topic instead of logs_topic because RabbitMQ STOMP plugin routes incoming messages
// to the amq.topic exchange.
func SendTopic(ch *amqp.Channel) error {
	var err error
	listenMethod := os.Getenv("LISTEN_METHOD")
	exchange := "logs_topic"
	if listenMethod == "rbmq_stomp" {
		// STOMP channels' queues are attached to amq.topic exchange
		exchange = "amq.topic"
	}
	if err = ch.Publish(
		exchange,
		"something.somewhere",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("hello world"),
		}); err != nil {
		return err
	}
	return nil
}

// SendQueue sends a message to a direct exchange with routing key "testing".
func SendQueue(conn *amqp.Connection, ch *amqp.Channel) error {
	var err error
	if ch, err = GetNewChannel(conn); err != nil {
		return err
	}
	if err = ch.Publish(
		"",
		"testing",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("hello"),
		}); err != nil {
		return err
	}
	return nil
}

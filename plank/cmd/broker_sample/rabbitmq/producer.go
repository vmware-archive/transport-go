package rabbitmq

import (
	"github.com/streadway/amqp"
	"os"
)

func GetNewConnection(url string) (conn *amqp.Connection, err error) {
	conn, err = amqp.Dial(url)
	return
}

func GetNewChannel(conn *amqp.Connection) (ch *amqp.Channel, err error) {
	ch, err = conn.Channel()
	return
}

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
			Body:        []byte("yo"),
		}); err != nil {
		return err
	}
	return nil
}

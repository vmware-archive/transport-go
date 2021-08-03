package main

import (
	"github.com/jooskim/plank/utils"
	"github.com/streadway/amqp"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672")
	if err != nil {
		utils.Log.Fatalln(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		utils.Log.Fatalln(err)
	}
	defer ch.Close()

	if err = ch.ExchangeDeclare(
		"logs_direct",
		"direct",
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
		"info",
		"logs_direct",
		false,
		nil); err != nil {
		utils.Log.Fatalln(err)
	}

	forever := make(chan struct{})
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		utils.Log.Fatalln(err)
	}

	go func() {
		for d := range msgs {
			utils.Log.Infof("[x] %s", d.Body)
		}
	}()

	<-forever
}

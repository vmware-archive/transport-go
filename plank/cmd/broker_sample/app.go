package main

import (
	"github.com/jooskim/plank/cmd/broker_sample/plank"
	"github.com/jooskim/plank/cmd/broker_sample/rabbitmq"
	"github.com/jooskim/plank/utils"
	"github.com/streadway/amqp"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	listenMethod := os.Getenv("LISTEN_METHOD")
	produceMessage := os.Getenv("PRODUCE_MESSAGE_ON_RABBITMQ")

	var ch *amqp.Channel
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL)

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

			// produce a message
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
		plank.ListenViaWS(c, true)
		break
	case "rbmq_amqp":
		rabbitmq.ListenViaAmqp(c)
		break
	case "rbmq_stomp":
		rabbitmq.ListenViaStomp(c)
	}
}

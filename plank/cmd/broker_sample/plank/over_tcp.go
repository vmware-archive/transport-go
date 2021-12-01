package plank

import (
	"fmt"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"os"
	"syscall"
	"time"
)

// ListenViaStomp listens to another Plank instance via TCP
func ListenViaStomp(c chan os.Signal) {
	brokerConn, err := connectToBroker()
	if err != nil {
		fmt.Println(fmt.Errorf("broker connection failed: %w", err))
		os.Exit(1)
	}

	// subscribe to topic simple-stream which keeps sending a random word at an interval
	sub, err := brokerConn.Subscribe("/topic/simple-stream")
	if err != nil {
		fmt.Println(fmt.Errorf("subscription failed: %w", err))
		os.Exit(1)
	}

	// let's disconnect after 10 seconds for the sake of an example
	disconnectInTime := 10 * time.Second
	tenSecTimer := time.NewTimer(disconnectInTime)

	go func() {
		for {
			select {
			case msg := <-sub.GetMsgChannel():
				// extract payload (of string type) from the Message object
				var payload string
				if err := msg.CastPayloadToType(&payload); err != nil {
					fmt.Printf("failed to cast payload: %s\n", err.Error())
					continue
				}
				fmt.Printf("msg body: %s\n", payload)
				break
			case <-tenSecTimer.C:
				fmt.Println(disconnectInTime.String() + " elapsed. Disconnecting")
				tenSecTimer.Stop()
				c <- syscall.SIGINT
				break
			}
		}
	}()

	<-c
}

// connectToBroker wraps the connection logic to the broker and returns the bridge connection object and an error
func connectToBroker() (bridge.Connection, error) {
	b := bus.GetBus()
	brokerConfig := &bridge.BrokerConnectorConfig{
		Username:     "guest",
		Password:     "guest",
		ServerAddr:   ":61613",
		HeartBeatOut: 30 * time.Second,
	}

	brokerConn, err := b.ConnectBroker(brokerConfig)
	if err != nil {
		return nil, err
	}

	fmt.Println("broker connected. broker ID:", brokerConn.GetId())
	return brokerConn, nil
}

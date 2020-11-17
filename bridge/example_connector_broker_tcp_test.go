// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge_test

import (
    "fmt"
    "github.com/vmware/transport-go/bridge"
    "github.com/vmware/transport-go/bus"
)

func Example_connectUsingBrokerViaTCP() {

    // get a reference to the event bus.
    b := bus.GetBus()

    // create a broker connector configuration, using WebSockets.
    // Make sure you have a STOMP TCP server running like RabbitMQ
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: ":61613"}

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        fmt.Printf("unable to connect, error: %e", err)
    }
    defer c.Disconnect()

    // subscribe to our demo simple-stream
    s, _ := c.Subscribe("/queue/sample")

    // set a counter
    n := 0

    // create a control chan
    done := make(chan bool)

    // listen for messages
    var consumer = func() {
        for {
            // listen for incoming messages from subscription.
            m := <-s.GetMsgChannel()
            n++

            // get byte array.
            d := m.Payload.([]byte)

            fmt.Printf("Message Received: %s\n", string(d))
            // listen for 5 messages then stop.
            if n >= 5 {
                break
            }
        }
        done <- true
    }

    // send messages
    var producer = func() {
        for i := 0; i < 5; i++ {
            c.SendMessage("/queue/sample", []byte(fmt.Sprintf("message: %d", i)))
        }
    }

    // listen for incoming messages on subscription for destination /queue/sample
    go consumer()

    // send some messages to the broker on destination /queue/sample
    go producer()

    // wait for messages to be processed.
    <-done
}

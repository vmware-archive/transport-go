// Copyright 2019 VMware Inc.
package main

import (
    "bifrost/bridge"
    "bifrost/bus"
    "bifrost/model"
    "encoding/json"
    "fmt"
)

func main() {
    //
    //bf := bus.GetBus()
    //channel := "some-channel"
    //bf.GetChannelManager().CreateChannel(channel)
    //
    //// listen for a single request on 'some-channel'
    //requestHandler, _ := bf.ListenRequestStream(channel)
    //requestHandler.Handle(
    //   func(msg *model.Message) {
    //       pingContent := msg.Payload.(string)
    //       fmt.Printf("\nPing: %s\n", pingContent)
    //
    //       // send a response back.
    //       bf.SendResponseMessage(channel, pingContent, msg.DestinationId)
    //   },
    //   func(err error) {
    //       // something went wrong...
    //   })
    //
    //// send a request to 'some-channel' and handle a single response
    //responseHandler, _ := bf.RequestOnce(channel, "Woo!")
    //responseHandler.Handle(
    //   func(msg *model.Message) {
    //       fmt.Printf("Pong: %s\n", msg.Payload.(string))
    //   },
    //   func(err error) {
    //       // something went wrong...
    //   })
    //// fire the request.
    //responseHandler.Fire()

    // get a reference to the event bus.
    b := bus.GetBus()

    // create a broker connector configuration, using WebSockets.
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: "appfabric.vmware.com:8090",
        WSPath:     "/fabric",
        UseWS:      true}

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        fmt.Printf("unable to connect, error: %e", err)
    }

    // subscribe to our demo simple-stream
    s, _ := c.Subscribe("/topic/simple-stream")

    // set a counter
    n := 0

    // create a control chan
    done := make(chan bool)

    var listener = func() {
        for {
            // listen for incoming messages from subscription.
            m := <-s.C

            // unmarshal message.
            r := &model.Response{}
            d := m.Payload.([]byte)
            json.Unmarshal(d, &r)
            fmt.Printf("Message Received: %s\n", r.Payload.(string))

            n++

            // listen for 5 messages then stop.
            if n >= 5 {
                break
            }
        }
        done <- true
    }

    // listen for incoming messages on subscription.
    go listener()

    <-done

    c.Disconnect()

}

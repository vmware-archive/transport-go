// Copyright 2019 VMware Inc.
package main

import (
    "bifrost/bridge"
    "bifrost/bus"
    "bifrost/model"
    "encoding/json"
    "fmt"
    "log"
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

    // get a pointer to the bus.
    b := bus.GetBus()

    // get a pointer to the channel manager
    cm := b.GetChannelManager()

    channel := "my-stream"
    cm.CreateChannel(channel)

    // create done signal
    var done = make(chan bool)

    // listen to stream of messages coming in on channel.
    h, err := b.ListenStream(channel)

    if err != nil {
        log.Panicf("unable to listen to channel stream, error: %e", err)
    }

    count := 0

    // listen for five messages and then exit, send a completed signal on channel.
    h.Handle(
        func(msg *model.Message) {

            // unmarshal the payload into a Response object (used by fabric services)
            r := &model.Response{}
            d := msg.Payload.([]byte)
            json.Unmarshal(d, &r)
            fmt.Printf("Stream Ticked: %s\n", r.Payload.(string))
            count++
            if count >=5 {
                done <- true
            }
        },
        func(err error) {
            log.Panicf("error received on channel %e", err)
        })

    // create a broker connector config, in this case, we will connect to the application fabric demo endpoint.
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: "appfabric.vmware.com",
        WSPath:     "/fabric",
        UseWS:      true}

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        log.Panicf("unable to connect to fabric, error: %e", err)
    }

    // mark our local channel as galactic and map it to our connection and the /topic/simple-stream service
    // running on appfabric.vmware.com
    err = cm.MarkChannelAsGalactic(channel, "/topic/simple-stream", c)
    if err != nil {
        log.Panicf("unable to map local channel to broker destination: %e", err)
    }

    // wait for done signal
    <-done

    // mark channel as local (unsubscribe from all mappings)
    err = cm.MarkChannelAsLocal(channel)
    if err != nil {
        log.Panicf("unable to unsubscribe, error: %e", err)
    }
    err = c.Disconnect()
    if err != nil {
        log.Panicf("unable to disconnect, error: %e", err)
    }
}

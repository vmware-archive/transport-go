# Transport - Golang

![Transport Post-merge pipeline](https://github.com/vmware/transport-go/workflows/Transport%20Post-merge%20pipeline/badge.svg)
[![codecov](https://codecov.io/gh/vmware/transport-go/branch/main/graph/badge.svg?token=BgZhsCCZ0k)](https://codecov.io/gh/vmware/transport-go)

Transport is a full stack, simple, fast, expandable application event bus for your applications.

### What does that mean?

Transport is an event bus, that allows application developers to build components that can talk to one another, really easily.

It provides a standardized and simple API, implemented in multiple languages, to allow any individual component inside your applications to talk to one another.

It really comes to life when you use it to send messages, requests, responses and events around your backend and front-end. Your Java or Golang backend can stream messages to your UI components, as if they were sitting right next to each other.

Channels can be extended to major brokers like Kafka or RabbitMQ, so Transport becomes an 'on/off-ramp' for your main sources of truth.

### [View Transport Golang Documentation](https://vmware.github.io/transport/golang)

#### [Transport Docs Repo](https://github.com/vmware/transport)

## Quick Start

To create an instance of the bus

```go
var transport EventBus = bus.GetBus()
```

The API is pretty simple.

```go
type EventBus interface {
    GetId() *uuid.UUID
    GetChannelManager() ChannelManager
    SendRequestMessage(channelName string, payload interface{}, destinationId *uuid.UUID) error
    SendResponseMessage(channelName string, payload interface{}, destinationId *uuid.UUID) error
    SendErrorMessage(channelName string, err error, destinationId *uuid.UUID) error
    ListenStream(channelName string) (MessageHandler, error)
    ListenStreamForDestination(channelName string, destinationId *uuid.UUID) (MessageHandler, error)
    ListenFirehose(channelName string) (MessageHandler, error)
    ListenRequestStream(channelName string) (MessageHandler, error)
    ListenRequestStreamForDestination(channelName string, destinationId *uuid.UUID) (MessageHandler, error)
    ListenRequestOnce(channelName string) (MessageHandler, error)
    ListenRequestOnceForDestination (channelName string, destinationId *uuid.UUID) (MessageHandler, error)
    ListenOnce(channelName string) (MessageHandler, error)
    ListenOnceForDestination(channelName string, destId *uuid.UUID) (MessageHandler, error)
    RequestOnce(channelName string, payload interface{}) (MessageHandler, error)
    RequestOnceForDestination(channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error)
    RequestStream(channelName string, payload interface{}) (MessageHandler, error)
    RequestStreamForDestination(channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error)
}
```

- All methods throw an `error` if the channel does not yet exist.

## Managing Channels

The `ChannelManager` interface on the `EventBus` interface facilitates all Channel operations.

```go
channelManager := bf.GetChannelManager()
```

The `ChannelManager` interface is pretty simple.

```go
type ChannelManager interface {
    Boot()
    CreateChannel(channelName string) *Channel
    DestroyChannel(channelName string)
    CheckChannelExists(channelName string) bool
    GetChannel(channelName string) (*Channel, error)
    GetAllChannels() map[string]*Channel
    SubscribeChannelHandler(channelName string, fn MessageHandlerFunction, runOnce bool) (*uuid.UUID, error)
    UnsubscribeChannelHandler(channelName string, id *uuid.UUID) error
    WaitForChannel(channelName string) error
}
```

### Creating Channels

The `CreateChannel` method will create a new channel with the name "some-channel". It will return a pointer to a
`Channel` object. However you don't need to hold on to that pointer if you dont want.

```go
channel := channelManager.CreateChanel("some-channel")
```

## Simple Example

A simple ping pong looks a little like this.

```go
// listen for a single request on 'some-channel'
tr := bus.GetBus()
channel := "some-channel"
tr.GetChannelManager().CreateChannel(channel)

// listen for a single request on 'some-channel'
requestHandler, _ := bf.ListenRequestStream(channel)
requestHandler.Handle(
   func(msg *model.Message) {
       pingContent := msg.Payload.(string)
       fmt.Printf("\nPing: %s\n", pingContent)

       // send a response back.
       tr.SendResponseMessage(channel, pingContent, msg.DestinationId)
   },
   func(err error) {
       // something went wrong...
   })

// send a request to 'some-channel' and handle a single response
responseHandler, _ := tr.RequestOnce(channel, "Woo!")
responseHandler.Handle(
   func(msg *model.Message) {
       fmt.Printf("Pong: %s\n", msg.Payload.(string))
   },
   func(err error) {
       // something went wrong...
   })
// fire the request.
responseHandler.Fire()
```

This will output:

```text
🌈 Transport booted with id [e495e5d5-2b72-46dd-8013-d49049bd4800]
Ping: Woo!
Pong: Woo!
```

## Example connecting to a message broker and using galactic channels

If you would like to connect the bus to a broker and start streaming stuff, you can run the local demo broker
by first building using `./build-transport.sh` and then starting the local broker (and a bunch of demo services) via `
./transport-go service`

Once running, this example will connect to the broker and starts streaming over a local channel that is mapped to the live
sample service that is broadcasting every few hundred milliseconds on `/topic/simple-stream`

```go
package main

import (
   "encoding/json"
   "fmt"
   "github.com/vmware/transport-go/bridge"
   "github.com/vmware/transport-go/bus"
   "github.com/vmware/transport-go/model"
   "log"
)

func main() {
   usingGalacticChannels()
}

func usingGalacticChannels() {

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

   // listen for ten messages and then exit, send a completed signal on channel.
   h.Handle(
      func(msg *model.Message) {

         // unmarshal the payload into a Response object (used by fabric services)
         r := &model.Response{}
         d := msg.Payload.([]byte)
         json.Unmarshal(d, &r)
         fmt.Printf("Stream Ticked: %s\n", r.Payload.(string))
         count++
         if count >=10 {
            done <- true
         }
      },
      func(err error) {
         log.Panicf("error received on channel %e", err)
      })

   // create a broker connector config, in this case, we will connect to the demo endpoint.
   config := &bridge.BrokerConnectorConfig{
      Username:   "guest",
      Password:   "guest",
      ServerAddr: "localhost:8090",
      UseWS:      true,
      WebSocketConfig: &bridge.WebSocketConfig{
         WSPath:    "/fabric",
      }}

   // connect to broker.
   c, err := b.ConnectBroker(config)
   if err != nil {
      log.Panicf("unable to connect to fabric, error: %e", err)
   }

   // mark our local channel as galactic and map it to our connection and the /topic/simple-stream service
   // running on localhost:8090
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
```

### [Read More Golang Documentation](https://vmware.github.io/transport/golang)

## Contributing

The transport-go project team welcomes contributions from the community. Before you start working with transport-go, please
read our [Developer Certificate of Origin](https://cla.vmware.com/dco). All contributions to this repository must be
signed as described on that page. Your signature certifies that you wrote the patch or have the right to pass it on
as an open-source patch. For more detailed information, refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## License
BSD-2-Clause

Copyright (c) 2016-2021, VMware, Inc.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

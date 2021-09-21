# Transport - Golang

![Transport Post-merge pipeline](https://github.com/vmware/transport-go/workflows/Transport%20Post-merge%20pipeline/badge.svg)
[![codecov](https://codecov.io/gh/vmware/transport-go/branch/main/graph/badge.svg?token=BgZhsCCZ0k)](https://codecov.io/gh/vmware/transport-go)

Transport is a full stack, simple, fast, expandable application event bus for your applications. This is the golang version.

### What does that mean?

Transport is an event bus, that allows application developers to build components that can talk to one another, really easily.
It provides a standardized and simple API, implemented in multiple languages, to allow any individual component inside your 
applications to talk to one another.

It really comes to life when you use it to send messages, requests, responses and events around your backend and front-end. 
Your backend can stream messages to your UI components, as if they were sitting right next to each other.

Channels can be extended to major brokers like Kafka or RabbitMQ, so Transport becomes an 'on/off-ramp' for your main sources of truth.

### Watch this quick video for an overview

[![Quick Transport Overview](https://img.youtube.com/vi/k-KDPtCQyls/0.jpg)](https://www.youtube.com/watch?v=k-KDPtCQyls)

## Getting Started

### [View All Transport (Golang) Documentation](https://transport-bus.io/golang)

#### Other versions of Transport

- [Transport TypeScript](https://github.com/vmware/transport-typescript)
- [Transport Java](https://github.com/vmware/transport-java)

### Transport Site
[https://transport-bus.io](https://transport-bus.io)

---
## Quick Start

Install transport 

```go
go get -u github.com/vmware/transport-go
```

To create an instance of the bus

```go
import 	"github.com/vmware/transport-go/bus"

var transport EventBus = bus.GetBus()
```
> Transport is a singleton, there is (should) only ever a single instance of the bus in your application.

----

## Managing / Creating Channels

The `ChannelManager` interface on the `EventBus` interface facilitates all Channel operations.

```go
channelManager := transport.GetChannelManager()
```

### Creating Channels

The `CreateChannel` method will create a new channel with the name "some-channel". It will return a pointer to a
`Channel` object. You don't need to hold on to that pointer if you don't want to. The channel will still exist.

```go
channel := channelManager.CreateChanel("some-channel")
```

## Simple Example

A simple ping pong looks a little like this.

```go
// listen for a single request on 'some-channel'
ts := bus.GetBus()
channel := "some-channel"
ts.GetChannelManager().CreateChannel(channel)

// listen for a single request on 'some-channel'
requestHandler, _ := ts.ListenRequestStream(channel)
requestHandler.Handle(
   func(msg *model.Message) {
       pingContent := msg.Payload.(string)
       fmt.Printf("\nPing: %s\n", pingContent)

       // send a response back.
       ts.SendResponseMessage(channel, pingContent, msg.DestinationId)
   },
   func(err error) {
       // something went wrong...
   })

// send a request to 'some-channel' and handle a single response
responseHandler, _ := ts.RequestOnce(channel, "Woo!")
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
---
## Connecting to a message broker and using galactic channels

You can see this all working live in some of our interactive demos for [Transport TypeScript](https://transport-bus.io/ts/examples/joke-service).
it shows Transport acting as both client and server, in which we use [Plank](https://github.com/vmware/transport-go/tree/main/plank) to run
services, and the UI subscribes to those services and talks to them.

We have a live and running instance of [Plank](https://github.com/vmware/transport-go/tree/main/plank) operating at
[transport-bus.io](https://transport-bus.io). You can try the example code below to use the sample 
[simple stream service](https://github.com/vmware/transport-go/blob/main/plank/services/simple-stream-service.go) and 
see how simple it is for yourself.

### Simple Stream Example

```go
import (
	"encoding/json"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"sync"
)

// get a pointer to the bus.
b := bus.GetBus()

// get a pointer to the channel manager
cm := b.GetChannelManager()

// create a broker connector config and connect to 
// transport-bus.io over WebSocket using TLS.
config := &bridge.BrokerConnectorConfig{
    Username:   "guest",            // not required for demo, but our API requires it.
    Password:   "guest",            // ^^ same.
    ServerAddr: "transport-bus.io", // our live broker running plank and demo services.
    UseWS:      true,               // connect over websockets
    WebSocketConfig: &bridge.WebSocketConfig{ // configure websocket
        WSPath: "/ws", // websocket endpoint
        UseTLS: true,
        // use TLS/HTTPS. When using TLS, you can supply your own TLSConfig value, or we can
        // generate a basic one for you if you leave TLSConfig empty. In most cases, 
        // you won't need to supply one.
    }}

// connect to transport-bus.io demo broker
c, err := b.ConnectBroker(config)
if err != nil {
    utils.Log.Fatalf("unable to connect to transport-bus.io, error: %v", err.Error())
}

// create a local channel on the bus.
myLocalChan := "my-stream"
cm.CreateChannel(myLocalChan)

// listen to stream of messages coming in on channel, a handler is returned 
// that allows you to add in lambdas that handle your success messages, and your errors.
handler, _ := b.ListenStream(myLocalChan)

// mark our local 'my-stream' myLocalChan as galactic and map it to our connection and 
// the /topic/simple-stream service
err = cm.MarkChannelAsGalactic(myLocalChan, "/topic/simple-stream", c)
if err != nil {
    utils.Log.Fatalf("unable to map local channel to broker destination: %e", err)
}

// collect the streamed values in a slice
var streamedValues []string

// create a wait group that will wait 10 times before completing.
var wg sync.WaitGroup
wg.Add(10)

// keep listening
handler.Handle(
    func(msg *model.Message) {

        // unmarshal the message payload into a model.Response object
        // this is a wrapper transport uses when being used as a server, 
        // it encapsulates a rich set of data
        // about the message, but you only really care about the payload (body)
        r := &model.Response{}
        d := msg.Payload.([]byte)
        err := json.Unmarshal(d, &r)
        if err != nil {
            utils.Log.Errorf("error unmarshalling request: %v", err.Error())
            return
        }
        // the value we want is in the payload of our model.Response
        value := r.Payload.(string)

        // log it and save it to our streamedValues
        utils.Log.Infof("stream ticked: %s", value)
        streamedValues = append(streamedValues, value)
        wg.Done()
    },
    func(err error) {
        utils.Log.Errorf("error received on channel: %e", err)
    })

// wait for 10 ticks of the stream, then we're done.
wg.Wait()

// close our handler, we're done.
handler.Close()

// mark channel as local (unsubscribe from all mappings)
err = cm.MarkChannelAsLocal(myLocalChan)
if err != nil {
    utils.Log.Fatalf("unable to unsubscribe, error: %e", err)
}

// disconnect
err = c.Disconnect()
if err != nil {
    utils.Log.Fatalf("unable to disconnect, error: %e", err)
}

// return what we got from the stream.
return streamedValues
```

You can [see this simple example here](https://github.com/vmware/transport-go/examples/simple_stream.go) 


### [Read More Golang Documentation](https://transport-bus.io/golang)

---

### [What is `plank`?](https://github.com/vmware/transport-go/tree/main/plank)

`plank` is 'just enough' of a platform for building just about anything you want. Run REST and Async APIs with the 
same code, build simple or complex services that can be exposed to any client in any manner. Talk over WebSockets and pub/sub
and streaming, or call the same APIs via REST mappings. `plank` can do it all. It's tiny, super-fast and runs on any platform. Runs in
just a few megabytes of memory, and can be compiled down to the same. It can be used for micro-services, daemons, agents, 
local UI helper applications... anything!

`plank` is only available in transport-go

[Take a look at plank](https://github.com/vmware/transport-go/tree/main/plank)

---
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

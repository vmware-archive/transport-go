# Bifröst for Go

[![pipeline status](https://gitlab.eng.vmware.com/bifrost/golang/badges/master/pipeline.svg)](https://gitlab.eng.vmware.com/bifrost/golang/commits/master)
[![coverage report](https://gitlab.eng.vmware.com/bifrost/golang/badges/master/coverage.svg)](https://gitlab.eng.vmware.com/bifrost/golang/commits/master)

## Using the Bifröst

To create an instance of the bus

```go
bf := bus.GetBus()
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
bf := bus.GetBus()
channel := "some-channel"
bf.GetChannelManager().CreateChannel(channel)

// listen for a single request on 'some-channel'
requestHandler, _ := bf.ListenRequestStream(channel)
requestHandler.Handle(
   func(msg *model.Message) {
       pingContent := msg.Payload.(string)
       fmt.Printf("\nPing: %s\n", pingContent)

       // send a response back.
       bf.SendResponseMessage(channel, pingContent, msg.DestinationId)
   },
   func(err error) {
       // something went wrong...
   })

// send a request to 'some-channel' and handle a single response
responseHandler, _ := bf.RequestOnce(channel, "Woo!")
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
🌈 Bifröst booted with id [e495e5d5-2b72-46dd-8013-d49049bd4800]
Ping: Woo!
Pong: Woo!
```

## Example connecting to message broker and using galactic channels

If you would like to connect the bus to a broker and start streaming stuff, it's quite simple. Here is an example
that connects to `appfabric.vmware.com` and starts streaming over a local channel that is mapped to the live
sample service that it broadcasting every few hundred milliseconds on `/topic/simple-stream`

```go
import (
    "bifrost/bridge"
    "bifrost/bus"
    "bifrost/model"
    "encoding/json"
    "fmt"
    "log"
)

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
```

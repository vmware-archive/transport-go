#Bifröst for Go

## Using

To create an instance of the bus

```go
bus := GetBus()
```

The API is pretty simple.

```go
type EventBus interface {
    GetId() *uuid.UUID
    GetChannelManager() ChannelManager
    SendRequestMessage(channelName string, payload interface{}, id *uuid.UUID) error
    SendResponseMessage(channelName string, payload interface{}, id *uuid.UUID) error
    SendErrorMessage(channelName string, err error, id *uuid.UUID) error
    ListenStream(channelName string) (MessageHandler, error)
    ListenFirehose(channelName string) (MessageHandler, error)
    ListenRequestStream(channelName string) (MessageHandler, error)
    ListenOnce(channelName string) (MessageHandler, error)
    RequestOnce(channelName string, payload interface{}) (MessageHandler, error)
    RequestStream(channelName string, payload interface{}) (MessageHandler, error)
}
```

- All methods throw an `error` if the channel does not yet exist.

##Managing Channels

The `ChannelManager` interface on the `EventBus` interface facilitates all Channel operations.

```go
channelManager := bus.GetChannelManager()
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

###Creating Channels

The `CreateChannel` method will create a new channel with the name "some-channel". It will return a pointer to a
`Channel` object. However you don't need to hold on to that pointer if you dont want.

```go
channel := channelManager.CreateChanel("some-channel")
```

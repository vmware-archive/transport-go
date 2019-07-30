package bus

import (
    "errors"
    "fmt"
    "github.com/google/uuid"
)

type EventBus interface {
    Init()
    GetChannelManager() ChannelManager
    SendRequestMessage(channelName string, payload interface{}) error
    ListenStream(channelName string, handler func(interface{})) error
}

type BifrostEventBus struct {
    ChannelManager ChannelManager
    Id             uuid.UUID
}

func (bus *BifrostEventBus) Init() {
    bus.Id = uuid.New()
    bus.ChannelManager = new(ChannelManagerImpl)
    bus.ChannelManager.Boot()
    fmt.Printf("🌈 Bifröst booted with id [%s]", bus.Id.String())
}

func (bus *BifrostEventBus) GetChannelManager() ChannelManager {
    return bus.ChannelManager
}

func (bus *BifrostEventBus) SendRequestMessage(channelName string, payload interface{}) error {
    channelManager := bus.ChannelManager
    if !channelManager.CheckChannelExists(channelName) {
        return errors.New(fmt.Sprintf("channelName %s does not exist", channelName))
    }
    channelObject, _ := channelManager.GetChannel(channelName)
    message := &Message{
        Id:        uuid.New(),
        Channel:   channelName,
        Payload:   payload,
        Direction: Request}
    message.Channel = channelName
    message.Direction = Request
    channelObject.Send(message)
    return nil
}

func (bus *BifrostEventBus) ListenStream(channelName string, handler func(interface{})) error {
    channelManager := bus.ChannelManager
    if !channelManager.CheckChannelExists(channelName) {
        return errors.New(fmt.Sprintf("channelName %s does not exist", channelName))
    }
    //channelObject, _ := channelManager.GetChannel(channelName)

    go func() {
        //message := <-channelObject.Stream
        //handler(message)
        //<-channelObject.Control
    }()
    return nil
}

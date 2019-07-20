package bus

import (
    //"autogen-sources/libautogen/dsp"
    "fmt"
    "github.com/google/uuid"
)

type EventBus interface {
    Init()
    GetChannelManager() *ChannelManager
    SendRequestMessage(channel string, payload interface{})
}

type BifrostEventBus struct {
    ChannelManager *ChannelManager
    Id uuid.UUID
}

func (bus *BifrostEventBus) Init() {
    bus.Id = uuid.New()
    bus.ChannelManager = new(ChannelManager)
    fmt.Printf("🌈 Bifröst booted with id [%s]", bus.Id.String())
}

func (bus *BifrostEventBus) GetChannelManager() *ChannelManager {
    return bus.ChannelManager
}

func (bus *BifrostEventBus) SendRequestMessage(channel string, payload interface{}) {

}

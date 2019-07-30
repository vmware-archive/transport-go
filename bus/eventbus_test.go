package bus

import (
    "testing"
)

type Dummy struct {
    Name string
}

var bus EventBus
var channelName string = "test-channel"
var end = make(chan bool)
var busChannel *Channel
var channelManager ChannelManager

func init() {

    bus = new(BifrostEventBus)
    bus.Init()
}

func createTestChannel() {
    channelManager = bus.GetChannelManager()
    busChannel = channelManager.CreateChannel(channelName)
}

func destroyTestChannel() {
    channelManager.DestroyChannel(channelName)
}

func TestEventBusBoot(t *testing.T) {
    if bus.GetChannelManager() == nil {
        t.Error("Bus cannot be booted")
    }
}

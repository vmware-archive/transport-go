// Copyright 2019 VMware Inc.
package model

const (
    ChannelCreatedEvt          int = 0
    ChannelDestroyedEvt        int = 1
    ChannelSubscriberJoinedEvt int = 2
    ChannelSubscriberLeftEvt   int = 3
    ChannelMessageEvt          int = 4
    ChannelErrorEvt            int = 5
    ChannelIsGalacticEvt       int = 6
    ChannelIsLocalEvt          int = 7
    BrokerConnectedEvt         int = 8
    BrokerDisconnected         int = 9
)

type MonitorEvent struct {
    EventType    int
    Message      *Message
    Channel     string
}

// Create a new monitor event
func NewMonitorEvent(evtType int, channel string, message *Message,) *MonitorEvent {
    return &MonitorEvent{EventType: evtType, Message: message, Channel: channel}
}

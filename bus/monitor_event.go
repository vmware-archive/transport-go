// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential
package bus

type MonitorEventType int32
const (
    ChannelCreatedEvt          MonitorEventType = iota
    ChannelDestroyedEvt
    ChannelSubscriberJoinedEvt
    ChannelSubscriberLeftEvt
    StoreCreatedEvt
    StoreDestroyedEvt
    StoreInitializedEvt
    BrokerSubscribedEvt
    BrokerUnsubscribedEvt
    FabricEndpointSubscribeEvt
    FabricEndpointUnsubscribeEvt
)

type MonitorEventHandler func(event *MonitorEvent)

type MonitorEvent struct {
    // Type of the event
    EventType MonitorEventType
    // The name of the channel or the store related to this event
    EntityName   string
    // Optional event data
    Data      interface{}
}

// Create a new monitor event
func NewMonitorEvent(evtType MonitorEventType, entityName string, data interface{} ) *MonitorEvent {
    return &MonitorEvent{EventType: evtType, Data: data, EntityName: entityName}
}

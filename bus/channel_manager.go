// Copyright 2019 VMware Inc.

package bus

import (
    "bifrost/model"
    "bifrost/util"
    "errors"
    "fmt"
    "github.com/google/uuid"
)

// ChannelManager interfaces controls all access to channels vis the bus.
type ChannelManager interface {
    CreateChannel(channelName string) *Channel
    DestroyChannel(channelName string)
    CheckChannelExists(channelName string) bool
    GetChannel(channelName string) (*Channel, error)
    GetAllChannels() map[string]*Channel
    SubscribeChannelHandler(channelName string, fn MessageHandlerFunction, runOnce bool) (*uuid.UUID, error)
    UnsubscribeChannelHandler(channelName string, id *uuid.UUID) error
    WaitForChannel(channelName string) error
    MarkChannelAsGalactic(channelName string, brokerDestination string) (err error)
    MarkChannelAsLocal(channelName string) (err error)
}

func NewBusChannelManager(bus EventBus) ChannelManager {
    manager := new(busChannelManager)
    manager.Channels = make(map[string]*Channel)
    manager.bus = bus.(*bifrostEventBus)
    manager.monitor = util.GetMonitor()
    return manager
}


type busChannelManager struct {
    Channels map[string]*Channel
    bus      *bifrostEventBus
    monitor  *util.MonitorStream
}

// Create a new Channel with the supplied Channel name. Returns pointer to new Channel object
func (manager *busChannelManager) CreateChannel(channelName string) *Channel {
    manager.monitor.SendMonitorEvent(util.ChannelCreatedEvt, channelName)
    manager.Channels[channelName] = NewChannel(channelName)
    return manager.Channels[channelName]
}

// Destroy a Channel and all the handlers listening on it.
func (manager *busChannelManager) DestroyChannel(channelName string) {
    util.GetMonitor().SendMonitorEvent(util.ChannelDestroyedEvt, channelName)
    delete(manager.Channels, channelName)
}

// Get a pointer to a Channel by name. Returns points, or error if no Channel is found.
func (manager *busChannelManager) GetChannel(channelName string) (*Channel, error) {
    if channel, ok := manager.Channels[channelName]; ok {
        return channel, nil
    } else {
        return nil, errors.New("Channel does not exist: " + channelName)
    }
}

// Get all channels currently open. Returns a map of Channel names and pointers to those Channel objects.
func (manager *busChannelManager) GetAllChannels() map[string]*Channel {
    return manager.Channels
}

// Check Channel exists, returns true if so.
func (manager *busChannelManager) CheckChannelExists(channelName string) bool {
    return manager.Channels[channelName] != nil
}

// Subscribe new handler lambda for Channel, bool flag runOnce determines if this is a single Fire handler.
// Returns UUID pointer, or error if there is no Channel by that name.
func (manager *busChannelManager) SubscribeChannelHandler(channelName string, fn MessageHandlerFunction, runOnce bool) (*uuid.UUID, error) {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    id := uuid.New()
    channel.subscribeHandler(fn,
        &channelEventHandler{callBackFunction: fn, runOnce: runOnce, uuid: &id})
    util.GetMonitor().SendMonitorEvent(util.ChannelSubscriberJoinedEvt, channelName)
    return &id, nil
}

// Unsubscribe a handler for a Channel event handler.
func (manager *busChannelManager) UnsubscribeChannelHandler(channelName string, uuid *uuid.UUID) error {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return err
    }
    found := false
    for i, handler := range channel.eventHandlers {
        if handler.uuid.ID() == uuid.ID() {
            channel.removeEventHandler(i)
            found = true
        }
    }
    if !found {
        return fmt.Errorf("no handler in Channel '%s' for uuid [%s]", channelName, uuid)
    }
    util.GetMonitor().SendMonitorEvent(util.ChannelSubscriberLeftEvt, channelName)
    return nil
}

func (manager *busChannelManager) WaitForChannel(channelName string) error {
    channel, _ := manager.GetChannel(channelName)
    if channel == nil {
        return fmt.Errorf("no such Channel as '%s'", channelName)
    }
    channel.wg.Wait()
    return nil
}

// Mark a channel as Galactic. This will map this channel to the supplied broker destination, if the broker connector
// is active and connected, this will result in a subscription to the broker destination being created. Returns
// an error if the channel does not exist.
func (manager *busChannelManager) MarkChannelAsGalactic(channelName string, dest string) (err error) {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return
    }
    channel.SetGalactic(dest)
    m := model.GenerateRequest(&model.MessageConfig{Payload: dest})                   // set the mapped destination as the payload
    go util.GetMonitor().SendMonitorEventData(util.ChannelIsGalacticEvt, channelName, m) // inform the monitor.
    return nil
}

// Mark a channel as Local. This will unmap the channel from the broker destination, and perform an unsubscribe
// operation if the broker connector is active and connected. Returns an error if the channel does not exist.
func (manager *busChannelManager) MarkChannelAsLocal(channelName string) (err error) {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return
    }
    channel.SetLocal()
    go util.GetMonitor().SendMonitorEvent(util.ChannelIsLocalEvt, channelName) // inform the monitor.
    return nil
}

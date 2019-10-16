// Copyright 2019 VMware Inc.

package bus

import (
    "go-bifrost/bridge"
    "go-bifrost/model"
    "go-bifrost/util"
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
    MarkChannelAsGalactic(channelName string, brokerDestination string, connection bridge.Connection) (err error)
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
    Channels        map[string]*Channel
    bus             *bifrostEventBus
    monitor         *util.MonitorStream
}

// Create a new Channel with the supplied Channel name. Returns pointer to new Channel object
func (manager *busChannelManager) CreateChannel(channelName string) *Channel {
    manager.monitor.SendMonitorEvent(util.ChannelCreatedEvt, channelName)
    manager.Channels[channelName] = NewChannel(channelName)
    return manager.Channels[channelName]
}

// Destroy a Channel and all the handlers listening on it.
func (manager *busChannelManager) DestroyChannel(channelName string) {
    manager.monitor.SendMonitorEvent(util.ChannelDestroyedEvt, channelName)
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
    channel.subscribeHandler(&channelEventHandler{callBackFunction: fn, runOnce: runOnce, uuid: &id})
    manager.monitor.SendMonitorEvent(util.ChannelSubscriberJoinedEvt, channelName)
    return &id, nil
}

// Unsubscribe a handler for a Channel event handler.
func (manager *busChannelManager) UnsubscribeChannelHandler(channelName string, uuid *uuid.UUID) error {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return err
    }
    found := channel.unsubscribeHandler(uuid)
    if !found {
        return fmt.Errorf("no handler in Channel '%s' for uuid [%s]", channelName, uuid)
    }
    manager.monitor.SendMonitorEvent(util.ChannelSubscriberLeftEvt, channelName)
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
func (manager *busChannelManager) MarkChannelAsGalactic(channelName string, dest string, conn bridge.Connection) (err error) {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return
    }

    // mark as galactic/
    channel.SetGalactic(dest)

    // create a galactic event
    pl := &galacticEvent{conn: conn, dest: dest}

    manager.handleGalacticChannelEvent(channelName, pl)
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

    // get rid of all broker connections.
    channel.removeBrokerConnections()

    manager.handleLocalChannelEvent(channelName)

    return nil
}

func (manager *busChannelManager) handleGalacticChannelEvent(channelName string, ge *galacticEvent) {
    ch, _ := manager.GetChannel(channelName)

    if ge.conn == nil {
        return
    }

    // check if channel is already subscribed on this connection
    if !ch.isBrokerSubscribedToDestination(ge.conn, ge.dest) {
        if sub, e := ge.conn.Subscribe(ge.dest); e == nil {

            // add broker connection to channel.
            ch.addBrokerConnection(ge.conn)

            m := model.GenerateResponse(&model.MessageConfig{Payload: ge.dest}) // set the mapped destination as the payload
            ch.addBrokerSubscription(ge.conn, sub)
            manager.monitor.SendMonitorEventData(util.BrokerSubscribedEvt, channelName, m)
            select {
            case ch.brokerMappedEvent <- true: // let channel watcher know, the channel is mapped
            default: // if no-one is listening, drop.
            }
        }
    }
}

func (manager *busChannelManager) handleLocalChannelEvent(channelName string) {
    ch, _ := manager.GetChannel(channelName)
    // loop through all the connections we have mapped, and subscribe!
    for _, s := range ch.brokerSubs {
        if e := s.s.Unsubscribe(); e == nil {
            ch.removeBrokerSubscription(s.s)
            m := model.GenerateResponse(&model.MessageConfig{Payload: s.s.GetDestination()}) // set the unmapped destination as the payload
            manager.monitor.SendMonitorEventData(util.BrokerUnsubscribedEvt, channelName, m)
            select {
            case ch.brokerMappedEvent <- false: // let channel watcher know, the channel is un-mapped
            default: // if no-one is listening, drop.
            }
        }
    }
    // get rid of all broker subscriptions on this channel.
    ch.removeBrokerConnections()
}

type galacticEvent struct {
    conn bridge.Connection
    dest string
}

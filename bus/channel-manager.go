// Copyright 2019 VMware Inc.

package bus

import (
    "errors"
    "fmt"
    "github.com/google/uuid"
)

// ChannelManager interfaces controls all access to channels vis the bus.
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

type busChannelManager struct {
    Channels map[string]*Channel
}

// Boot up the channel manager
func (manager *busChannelManager) Boot() {
    manager.Channels = make(map[string]*Channel)
}

// Create a new channel with the supplied channel name. Returns pointer to new Channel object
func (manager *busChannelManager) CreateChannel(channelName string) *Channel {
    manager.Channels[channelName] = NewChannel(channelName)
    return manager.Channels[channelName]
}

// Destroy a channel and all the handlers listening on it.
func (manager *busChannelManager) DestroyChannel(channelName string) {
    delete(manager.Channels, channelName)
}

// Get a pointer to a channel by name. Returns points, or error if no Channel is found.
func (manager *busChannelManager) GetChannel(channelName string) (*Channel, error) {
    if channel, ok := manager.Channels[channelName]; ok {
        return channel, nil
    } else {
        return nil, errors.New("Channel does not exist: " + channelName)
    }
}

// Get all channels currently open. Returns a map of channel names and pointers to those Channel objects.
func (manager *busChannelManager) GetAllChannels() map[string]*Channel {
    return manager.Channels
}

// Check channel exists, returns true if so.
func (manager *busChannelManager) CheckChannelExists(channelName string) bool {
    return manager.Channels[channelName] != nil
}

// Subscribe new handler lambda for channel, bool flag runOnce determines if this is a single Fire handler.
// Returns UUID pointer, or error if there is no channel by that name.
func (manager *busChannelManager) SubscribeChannelHandler(channelName string, fn MessageHandlerFunction, runOnce bool) (*uuid.UUID, error) {
    channel, err := manager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }

    id := uuid.New()
    channel.subscribeHandler(fn,
        &channelEventHandler{callBackFunction: fn, runOnce: runOnce, uuid: id})

    return &id, nil
}

// Unsubscribe a handler for a channel event handler.
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
        return fmt.Errorf("no handler in channel '%s' for uuid [%s]", channelName, uuid)
    }
    return nil
}

func (manager *busChannelManager) WaitForChannel(channelName string) error {
    channel, _ := manager.GetChannel(channelName)
    if channel == nil {
        return fmt.Errorf("no such channel as '%s'", channelName)
    }
    channel.wg.Wait()
    return nil
}


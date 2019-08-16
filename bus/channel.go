// Copyright 2019 VMware Inc.

package bus

import (
    "bifrost/bridge"
    "bifrost/model"
    "github.com/google/uuid"
    "sync"
)

// Channel represents the stream and the subscribed event handlers waiting for ticks on the stream
type Channel struct {
    Name          string `json:"string"`
    eventHandlers []*channelEventHandler
    galactic      bool
    private       bool
    channelLock   sync.Mutex
    wg            sync.WaitGroup
    brokerSubs    map[*uuid.UUID]*bridge.Subscription
}

// Create a new Channel with the supplied Channel name. Returns a pointer to that Channel.
func NewChannel(channelName string) *Channel {
    c := &Channel{
        Name:          channelName,
        eventHandlers: []*channelEventHandler{},
        channelLock:   sync.Mutex{},
        galactic:      false,
        private:       false,
        wg:            sync.WaitGroup{},
        brokerSubs:    make(map[*uuid.UUID]*bridge.Subscription)}
    return c
}

// Mark the Channel as private
func (channel *Channel) SetPrivate(private bool) {
    channel.private = private
}

// Mark the Channel as galactic
func (channel *Channel) SetGalactic(galactic bool) {
    channel.galactic = galactic
}

// Returns true is the Channel is marked as galactic
func (channel *Channel) IsGalactic() bool {
    return channel.galactic
}

// Returns true if the Channel is marked as private
func (channel *Channel) IsPrivate() bool {
    return channel.private
}

// Send a new message on this Channel, to all event handlers.
func (channel *Channel) Send(message *model.Message) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()
    if eventHandlers := channel.eventHandlers; len(eventHandlers) > 0 {

        // if a handler is run once only, then the slice will be mutated mid cycle.
        // copy slice to ensure that removed handler is still fired.
        handlerDuplicate := make([]*channelEventHandler, 0, len(eventHandlers))
        handlerDuplicate = append(handlerDuplicate, eventHandlers...)
        for n, eventHandler := range handlerDuplicate {
            if eventHandler.runOnce && eventHandler.hasRun {
                channel.removeEventHandler(n) // remove from slice.
            }
            channel.wg.Add(1)
            go channel.sendMessageToHandler(eventHandler, message)

        }
    }
}

// Check if the Channel has any registered subscribers
func (channel *Channel) ContainsHandlers() bool {
    return len(channel.eventHandlers) > 0
}

// Send message to handler function
func (channel *Channel) sendMessageToHandler(handler *channelEventHandler, message *model.Message) {
    handler.callBackFunction(message)
    channel.wg.Done()
}

// Subscribe a new handler function.
func (channel *Channel) subscribeHandler(fn MessageHandlerFunction, handler *channelEventHandler) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()
    channel.eventHandlers = append(channel.eventHandlers, handler)
}

// Remove handler function from being subscribed to the Channel.
func (channel *Channel) removeEventHandler(index int) {
    numHandlers := len(channel.eventHandlers)
    if numHandlers <= 0 {
        return
    }
    if index >= numHandlers {
        return
    }

    // delete from event handler slice.
    copy(channel.eventHandlers[index:], channel.eventHandlers[index+1:])
    channel.eventHandlers[numHandlers-1] = nil
    channel.eventHandlers = channel.eventHandlers[:numHandlers-1]
}

func (channel *Channel) addBrokerSubscription(sub *bridge.Subscription) {
    channel.brokerSubs[sub.Id] = sub
}

func (channel *Channel) removeBrokerSubscription(sub *bridge.Subscription) {
    delete(channel.brokerSubs, sub.Id)
}



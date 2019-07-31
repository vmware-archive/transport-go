// Copyright 2019 VMware Inc.

package bus

import (
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
}

// Create a new channel with the supplied channel name. Returns a pointer to that channel.
func NewChannel(channelName string) *Channel {
    c := &Channel{
        Name:          channelName,
        eventHandlers: []*channelEventHandler{},
        channelLock:   sync.Mutex{},
        galactic:      false,
        private:       false,
        wg:            sync.WaitGroup{}}
    return c
}

// Mark the channel as private
func (channel *Channel) SetPrivate(private bool) {
    channel.private = private
}

// Mark the channel as galactic
func (channel *Channel) SetGalactic(galactic bool) {
    channel.galactic = galactic
}

// Returns true is the channel is marked as galactic
func (channel *Channel) IsGalactic() bool {
    return channel.galactic
}

// Returns true if the channel is marked as private
func (channel *Channel) IsPrivate() bool {
    return channel.private
}

// Send a new message on this channel, to all event handlers.
func (channel *Channel) Send(message *Message) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()
    if eventHandlers := channel.eventHandlers; len(eventHandlers) > 0 {

        // if a handler is run once only, then the slice will be mutated mid cycle.
        // copy slice to ensure that removed handler is still fired.
        handlerDuplicate := make([]*channelEventHandler, 0, len(eventHandlers))
        handlerDuplicate = append(handlerDuplicate, eventHandlers...)
        for n, eventHandler := range handlerDuplicate {
            if eventHandler.runOnce {
                channel.removeEventHandler(n) // remove from slice.
            }
            channel.wg.Add(1)
            go channel.sendMessageToHandler(eventHandler, message)

        }
    }
}

// Check if the channel has any registered subscribers
func (channel *Channel) ContainsHandlers() bool {
    return len(channel.eventHandlers) > 0
}

// Send message to handler function
func (channel *Channel) sendMessageToHandler(handler *channelEventHandler, message *Message) {
    defer channel.wg.Done()
    handler.callBackFunction(message)

}

// Subscribe a new handler function.
func (channel *Channel) subscribeHandler(fn MessageHandlerFunction, handler *channelEventHandler) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()
    channel.eventHandlers = append(channel.eventHandlers, handler)
}

// Remove handler function from being subscribed to the channel.
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

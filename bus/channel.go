// Copyright 2019 VMware Inc.

package bus

import (
    "go-bifrost/bridge"
    "go-bifrost/model"
    "sync"
    "sync/atomic"
)

// Channel represents the stream and the subscribed event handlers waiting for ticks on the stream
type Channel struct {
    Name                      string `json:"string"`
    eventHandlers             []*channelEventHandler
    galactic                  bool
    galacticMappedDestination string
    private                   bool
    channelLock               sync.Mutex
    wg                        sync.WaitGroup
    brokerSubs                []*connectionSub
    brokerConns               []*bridge.Connection
    brokerMappedEvent         chan bool
}

// Create a new Channel with the supplied Channel name. Returns a pointer to that Channel.
func NewChannel(channelName string) *Channel {
    c := &Channel{
        Name:              channelName,
        eventHandlers:     []*channelEventHandler{},
        channelLock:       sync.Mutex{},
        galactic:          false,
        private:           false,
        wg:                sync.WaitGroup{},
        brokerMappedEvent: make(chan bool, 10),
        brokerConns:       []*bridge.Connection{},
        brokerSubs:        []*connectionSub{}}
    return c
}

// Mark the Channel as private
func (channel *Channel) SetPrivate(private bool) {
    channel.private = private
}

// Mark the Channel as galactic
func (channel *Channel) SetGalactic(mappedDestination string) {
    channel.galactic = true
    channel.galacticMappedDestination = mappedDestination
}

// Mark the Channel as local
func (channel *Channel) SetLocal() {
    channel.galactic = false
    channel.galacticMappedDestination = ""
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
            if eventHandler.runOnce && atomic.LoadInt64(&eventHandler.runCount) > 0 {
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
    atomic.AddInt64(&handler.runCount, 1)
    channel.wg.Done()
}

// Subscribe a new handler function.
func (channel *Channel) subscribeHandler(handler *channelEventHandler) {
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

func (channel *Channel) listenToBrokerSubscription(sub *bridge.Subscription) {
    for {
        msg, m := <-sub.C
        if m {
            channel.Send(msg)
        } else {
            break
        }
    }
}

func (channel *Channel) isBrokerSubscribed(sub *bridge.Subscription) bool {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()

    for _, cs := range channel.brokerSubs {
        if sub.Id.ID() == cs.s.Id.ID() {
            return true
        }
    }
    return false
}

func (channel *Channel) isBrokerSubscribedToDestination(c *bridge.Connection, dest string) bool {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()

    for _, cs := range channel.brokerSubs {
        if cs.s != nil && cs.s.Destination == dest && cs.c != nil && cs.c.Id == c.Id {
            return true
        }
    }
    return false
}

func (channel *Channel) addBrokerConnection(c *bridge.Connection) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()

    for _, brCon := range channel.brokerConns {
        if brCon.Id == c.Id {
            return
        }
    }

    channel.brokerConns = append(channel.brokerConns, c)
}

func (channel *Channel) removeBrokerConnections() {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()

    channel.brokerConns = []*bridge.Connection{}
}

func (channel *Channel) addBrokerSubscription(conn *bridge.Connection, sub *bridge.Subscription) {
    cs := &connectionSub{c: conn, s: sub}

    channel.channelLock.Lock()
    channel.brokerSubs = append(channel.brokerSubs, cs)
    channel.channelLock.Unlock()

    go channel.listenToBrokerSubscription(sub)
}

func (channel *Channel) removeBrokerSubscription(sub *bridge.Subscription) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()

    for i, cs := range channel.brokerSubs {
        if sub.Id.ID() == cs.s.Id.ID() {
            channel.brokerSubs = removeSub(channel.brokerSubs, i)
        }
    }
}

func removeSub(s []*connectionSub, i int) []*connectionSub {
    s[len(s)-1], s[i] = s[i], s[len(s)-1]
    return s[:len(s)-1]
}

type connectionSub struct {
    c *bridge.Connection
    s *bridge.Subscription
}

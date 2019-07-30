package bus

import (
    "fmt"
    "reflect"
    "sync"
)

type Command int

const (
    Close Command = 0
    Subscribed Command = 1
    Unsubscribed Command = 2
)

type ChannelControl struct {
    Command Command
}

type Channel struct {
    Name                string `json:"string"`
    eventHandlers       []*channelEventHandler
    channelLock         sync.Mutex
    wg                  sync.WaitGroup
}

// create a new channel
func NewChannel(channelName string) *Channel {
    c := &Channel{
        Name:           channelName,
        eventHandlers:  []*channelEventHandler{},
        channelLock:    sync.Mutex{},
        wg:             sync.WaitGroup{} }
    return c
}

// send a message on a channel
func (channel *Channel) Send(message *Message) {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()
    if eventHandlers := channel.eventHandlers; len(eventHandlers) > 0 {
        handlerDuplicate := make([]*channelEventHandler, 0, len(eventHandlers))
        handlerDuplicate = append(handlerDuplicate, eventHandlers...)
        for _, eventHandler := range handlerDuplicate {
            if eventHandler.runOnce {
             // TODO: add this.
            }
            channel.wg.Add(1)
            go channel.sendMessageToHandler(eventHandler, message)

        }
    }

}

// send message to handler
func (channel *Channel) sendMessageToHandler(handler *channelEventHandler, message *Message) {
    defer channel.wg.Done()
    handler.callBackFunction.Call([]reflect.Value{ reflect.ValueOf(message) })
}

func (channel *Channel) ContainsHandlers() bool {
   return len(channel.eventHandlers) > 0
}

// subscribe a new handler.
func (channel *Channel) subscribeHandler(fn interface{}, handler *channelEventHandler) error {
    channel.channelLock.Lock()
    defer channel.channelLock.Unlock()
    if !(reflect.TypeOf(fn).Kind() == reflect.Func) {
        return fmt.Errorf(`closure %s is not a function, can't be registered`, reflect.TypeOf(fn).Kind())
    }
    channel.eventHandlers = append(channel.eventHandlers, handler)
    return nil
}

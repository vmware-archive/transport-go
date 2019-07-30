package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "reflect"
    "testing"
)

var testChannelName string = "testing"

func TestCheckChannelCreation(t *testing.T) {

    channel := NewChannel(testChannelName)
    assert.Empty(t, channel.eventHandlers)

}

func TestSubscribeHandler(t *testing.T) {

    channel := NewChannel(testChannelName)
    handler := func(message string) {}
    err := channel.subscribeHandler(handler,
        &channelEventHandler{reflect.ValueOf(handler), false})

    if err != nil {
        assert.Fail(t, "unable to subscribe handler")
    }

    assert.Equal(t, 1, len(channel.eventHandlers))

    err = channel.subscribeHandler(handler,
        &channelEventHandler{reflect.ValueOf(handler), false})

    if err != nil {
        assert.Fail(t, "unable to subscribe second handler")
    }

    assert.Equal(t, 2, len(channel.eventHandlers))
}

func TestHandlerCheck(t *testing.T) {
    channel := NewChannel(testChannelName)

    assert.False(t, channel.ContainsHandlers())

    handler := func(message string) {}
    err := channel.subscribeHandler(handler,
        &channelEventHandler{reflect.ValueOf(handler), false})

    if err != nil {
        assert.Fail(t, "unable to subscribe handler")
    }

    assert.True(t, channel.ContainsHandlers())
}

func TestSendMessage(t *testing.T) {

    channel := NewChannel(testChannelName)
    handler := func(message *Message) {
        assert.Equal(t, message.Payload.(string), "pickled eggs")
        assert.Equal(t, message.Channel, testChannelName)
    }

    channel.subscribeHandler(handler,
        &channelEventHandler{reflect.ValueOf(handler), false})

    var message = &Message{
        Id:        uuid.New(),
        Payload:   "pickled eggs",
        Channel:   testChannelName,
        Direction: Request}

    channel.Send(message)
    channel.wg.Wait()
}

func TestSendMultipleMessages(t *testing.T) {

    channel := NewChannel(testChannelName)
    counter := 0
    handler := func(message *Message) {
        assert.Equal(t, message.Payload.(string), "chewy louie")
        assert.Equal(t, message.Channel, testChannelName)
        counter++
    }

    channel.subscribeHandler(handler,
        &channelEventHandler{reflect.ValueOf(handler), false})

    var message = &Message{
        Id:        uuid.New(),
        Payload:   "chewy louie",
        Channel:   testChannelName,
        Direction: Request}

    channel.Send(message)
    channel.Send(message)
    channel.Send(message)
    channel.wg.Wait()
    assert.Equal(t, 3, counter)
}

func TestMultiHandlerSingleMessage(t *testing.T) {

    channel := NewChannel(testChannelName)
    counterA, counterB, counterC := 0, 0, 0
    handlerA := func(message *Message) {
        counterA++
    }
    handlerB := func(message *Message) {
        counterB++
    }
    handlerC := func(message *Message) {
        counterC++
    }

    channel.subscribeHandler(handlerA,
        &channelEventHandler{reflect.ValueOf(handlerA), false})

    channel.subscribeHandler(handlerB,
        &channelEventHandler{reflect.ValueOf(handlerB), false})

    channel.subscribeHandler(handlerC,
        &channelEventHandler{reflect.ValueOf(handlerC), false})

    var message = &Message{
        Id:        uuid.New(),
        Payload:   "late night munchies",
        Channel:   testChannelName,
        Direction: Request}

    channel.Send(message)
    channel.Send(message)
    channel.Send(message)
    channel.wg.Wait()
    value := counterA + counterB + counterC

    assert.Equal(t, 9, value)
}


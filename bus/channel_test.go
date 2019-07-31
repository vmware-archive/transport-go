// Copyright 2019 VMware Inc.

package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

var testChannelName string = "testing"

func TestChannel_CheckChannelCreation(t *testing.T) {

    channel := NewChannel(testChannelName)
    assert.Empty(t, channel.eventHandlers)

}

func TestChannel_SubscribeHandler(t *testing.T) {

    channel := NewChannel(testChannelName)
    handler := func(*Message) {}
    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: uuid.New()})

    assert.Equal(t, 1, len(channel.eventHandlers))

    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: uuid.New()})



    assert.Equal(t, 2, len(channel.eventHandlers))
}

func TestChannel_HandlerCheck(t *testing.T) {
    channel := NewChannel(testChannelName)

    assert.False(t, channel.ContainsHandlers())

    handler := func(*Message) {}
    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: uuid.New()})


    assert.True(t, channel.ContainsHandlers())
}

func TestChannel_SendMessage(t *testing.T) {

    channel := NewChannel(testChannelName)
    handler := func(message *Message) {
        assert.Equal(t, message.Payload.(string), "pickled eggs")
        assert.Equal(t, message.Channel, testChannelName)
    }

    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: uuid.New()})

    id := uuid.New()
    var message = &Message{
        Id:        &id,
        Payload:   "pickled eggs",
        Channel:   testChannelName,
        Direction: Request}

    channel.Send(message)
    channel.wg.Wait()
}

func TestChannel_SendMultipleMessages(t *testing.T) {

    channel := NewChannel(testChannelName)
    counter := 0
    handler := func(message *Message) {
        assert.Equal(t, message.Payload.(string), "chewy louie")
        assert.Equal(t, message.Channel, testChannelName)
        counter++
    }

    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: uuid.New()})
    id := uuid.New()
    var message = &Message{
        Id:        &id,
        Payload:   "chewy louie",
        Channel:   testChannelName,
        Direction: Request}

    channel.Send(message)
    channel.Send(message)
    channel.Send(message)
    channel.wg.Wait()
    assert.Equal(t, 3, counter)
}

func TestChannel_SendMultipleMessagesSingleRunHandler(t *testing.T) {

    channel := NewChannel(testChannelName)
    counter := 0
    handler := func(message *Message) {
        assert.Equal(t, message.Payload.(string), "chewy louie")
        assert.Equal(t, message.Channel, testChannelName)
        counter++
    }

    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: true, uuid: uuid.New()})
    id := uuid.New()
    var message = &Message{
        Id:        &id,
        Payload:   "chewy louie",
        Channel:   testChannelName,
        Direction: Request}

    channel.Send(message)
    channel.Send(message)
    channel.Send(message)
    channel.wg.Wait()
    assert.Equal(t, 1, counter)
}

func TestChannel_MultiHandlerSingleMessage(t *testing.T) {

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
        &channelEventHandler{callBackFunction: handlerA, runOnce: false, uuid: uuid.New()})

    channel.subscribeHandler(handlerB,
        &channelEventHandler{callBackFunction: handlerB, runOnce: false, uuid: uuid.New()})

    channel.subscribeHandler(handlerC,
        &channelEventHandler{callBackFunction: handlerC, runOnce: false, uuid: uuid.New()})

    id := uuid.New()
    var message = &Message{
        Id:        &id,
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

func TestChannel_Privacy(t *testing.T) {
    channel := NewChannel(testChannelName)
    assert.False(t, channel.private)
    channel.SetPrivate(true)
    assert.True(t, channel.IsPrivate())
}

func TestChannel_ChannelGalactic (t *testing.T) {
    channel := NewChannel(testChannelName)
    assert.False(t, channel.galactic)
    channel.SetGalactic(true)
    assert.True(t, channel.IsGalactic())
}

func TestChannel_RemoveEventHandler(t *testing.T) {
    channel := NewChannel(testChannelName)
    handlerA:= func(message *Message) {}
    handlerB:= func(message *Message) {}

    idA := uuid.New()
    idB := uuid.New()

    channel.subscribeHandler(handlerA,
        &channelEventHandler{callBackFunction: handlerA, runOnce: false, uuid: idA})

    channel.subscribeHandler(handlerB,
        &channelEventHandler{callBackFunction: handlerB, runOnce: false, uuid: idB})

    assert.Len(t, channel.eventHandlers, 2)

    // remove the first handler (A)
    channel.removeEventHandler(0)
    assert.Len(t, channel.eventHandlers, 1)
    assert.Equal(t, idB.String(), channel.eventHandlers[0].uuid.String())

    // remove the second handler B)
    channel.removeEventHandler(0)
    assert.True(t, len(channel.eventHandlers) == 0)

}

func TestChannel_RemoveEventHandlerNoHandlers(t *testing.T) {
    channel := NewChannel(testChannelName)

    channel.removeEventHandler(0)
    assert.Len(t, channel.eventHandlers, 0)
}

func TestChannel_RemoveEventHandlerOOBIndex(t *testing.T) {
    channel := NewChannel(testChannelName)

    handler := func(*Message) {}
    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: uuid.New()})

    channel.removeEventHandler(999)
    assert.Len(t, channel.eventHandlers, 1)
}

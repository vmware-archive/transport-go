// Copyright 2019 VMware Inc.
package bus

import (
    "errors"
    "fmt"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

var evtBusTest EventBus
var evtbusTestChannelName string = "test-channel"
var evtbusTestManager ChannelManager

func init() {
    evtBusTest = GetBus()
}

func createTestChannel() *Channel {
    evtbusTestManager = evtBusTest.GetChannelManager()
    return evtbusTestManager.CreateChannel(evtbusTestChannelName)
}

func destroyTestChannel() {
    evtbusTestManager.DestroyChannel(evtbusTestChannelName)
}

func TestEventBus_Boot(t *testing.T) {

    bus1 := GetBus()
    bus2 := GetBus()
    bus3 := GetBus()

    assert.EqualValues(t, bus1.GetId(), bus2.GetId())
    assert.EqualValues(t, bus2.GetId(), bus3.GetId())
    assert.NotNil(t, evtBusTest.GetChannelManager())
}

func TestEventBus_SendResponseMessageNoChannel(t *testing.T) {
    err := evtBusTest.SendResponseMessage("channel-not-here", "hello melody", nil)
    assert.NotNil(t, err)
}

func TestEventBus_SendRequestMessageNoChannel(t *testing.T) {
    err := evtBusTest.SendRequestMessage("channel-not-here", "hello melody", nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenStream(t *testing.T) {
    createTestChannel()
    handler, err := evtBusTest.ListenStream(evtbusTestChannelName)
    assert.Nil(t, err)
    assert.NotNil(t, handler)
    count := 0
    handler.Handle(
        func(msg *Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 500; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "hello melody", nil)

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, nil)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1, nil)
    }
    destroyTestChannel()
}

func TestEventBus_ListenStreamNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenStream("missing-channel")
    assert.NotNil(t, err)
}

func TestEventBus_ListenOnce(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenOnce(evtbusTestChannelName)
    count := 0
    handler.Handle(
        func(msg *Message) {
            count++
        },
        func(err error) {})

    for i := 0; i < 300; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 0, handler.GetId())

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, handler.GetId())
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1, handler.GetId())
    }
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_ListenOnceNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenOnce("missing-channel")
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestStream(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenRequestStream(evtbusTestChannelName)
    count := 0
    handler.Handle(
        func(msg *Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 300; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody", handler.GetId())

        // send responses to make sure we're only getting requests
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion if picked up", handler.GetId())
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion again", handler.GetId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 300, count)
    destroyTestChannel()
}

func TestEventBus_ListenRequestOnce(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenRequestOnce(evtbusTestChannelName)
    count := 0
    handler.Handle(
        func(msg *Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 5; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody", handler.GetId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_ListenRequestStreamNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenRequestStream("missing-channel")
    assert.NotNil(t, err)
}

func TestEventBus_TestErrorMessageHandling(t *testing.T) {
    createTestChannel()

    err := evtBusTest.SendErrorMessage("invalid-channel", errors.New("something went wrong"), nil)
    assert.NotNil(t, err)

    handler, _ := evtBusTest.ListenStream(evtbusTestChannelName)
    countError := 0
    handler.Handle(
        func(msg *Message) {},
        func(err error) {
            assert.Errorf(t, err, "something went wrong")
            countError++
        })

    for i := 0; i < 5; i++ {
        err := errors.New("something went wrong")
        evtBusTest.SendErrorMessage(evtbusTestChannelName, err, handler.GetId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 5, countError)
    destroyTestChannel()
}

func TestEventBus_ListenFirehose(t *testing.T) {
    createTestChannel()
    counter := 0

    responseHandler, _ := evtBusTest.ListenFirehose(evtbusTestChannelName)
    responseHandler.Handle(
        func(msg *Message) {
            counter++
        },
        func(err error) {
            counter++
        })

    for i := 0; i < 5; i++ {
        err := errors.New("something went wrong")
        evtBusTest.SendErrorMessage(evtbusTestChannelName, err, responseHandler.GetId())
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, responseHandler.GetId())
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 1, responseHandler.GetId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.True(t, counter >= 14) // sometimes the last tick is missed, go routine completes before counter in handler ticks.
}

func TestEventBus_ListenFirehoseNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenFirehose("missing-channel")
    assert.NotNil(t, err)
}

func TestEventBus_RequestOnce(t *testing.T) {
    createTestChannel()

    handler, _ := evtBusTest.ListenRequestStream(evtbusTestChannelName)
    handler.Handle(
        func(msg *Message) {
            assert.Equal(t, "who is a pretty baby?", msg.Payload.(string))
            evtBusTest.SendResponseMessage(evtbusTestChannelName, "why melody is of course", msg.Id)
        },
        func(err error) {})

    count := 0
    responseHandler, _ := evtBusTest.RequestOnce(evtbusTestChannelName, "who is a pretty baby?")
    responseHandler.Handle(
        func(msg *Message) {
            assert.Equal(t, "why melody is of course", msg.Payload.(string))
            count++
        },
        func(err error) {})

    responseHandler.Fire()
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_RequestStream(t *testing.T) {
    channel := createTestChannel()
    handler := func(message *Message) {
        if message.Direction == Request {
            assert.Equal(t, "who has the cutest laugh?", message.Payload.(string))
            config := buildConfig(channel.Name, "why melody does of course", message.Id)

            // fire a few times, ensure that the handler only ever picks up a single response.
            for i := 0; i < 5; i++ {
                channel.Send(generateResponse(config))
            }
        }
    }
    id := uuid.New()
    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: false, uuid: id})

    count := 0
    responseHandler, _ := evtBusTest.RequestStream(evtbusTestChannelName, "who has the cutest laugh?")
    responseHandler.Handle(
        func(msg *Message) {
            assert.Equal(t, "why melody does of course", msg.Payload.(string))
            count++
        },
        func(err error) {})

    responseHandler.Fire()
    assert.Equal(t, 5, count)
    destroyTestChannel()
}

func TestEventBus_HandleSingleRunError(t *testing.T) {
    channel := createTestChannel()
    handler := func(message *Message) {
        if message.Direction == Request {
            config := buildError(channel.Name, fmt.Errorf("whoops!"))

            // fire a few times, ensure that the handler only ever picks up a single response.
            for i := 0; i < 5; i++ {
                channel.Send(generateError(config))
            }
        }
    }
    id := uuid.New()
    channel.subscribeHandler(handler,
        &channelEventHandler{callBackFunction: handler, runOnce: true, uuid: id})

    count := 0
    responseHandler, _ := evtBusTest.RequestOnce(evtbusTestChannelName, 0)
    responseHandler.Handle(
        func(msg *Message) {},
        func(err error) {
            assert.Error(t, err, "whoops!")
            count++
        })

    responseHandler.Fire()
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_RequestOnceNoChannel(t *testing.T) {
    _, err := evtBusTest.RequestOnce("missing-channel", 0)
    assert.NotNil(t, err)
}

func TestEventBus_HandlerWithoutRequestToFire(t *testing.T) {
    createTestChannel()
    responseHandler, _ := evtBusTest.ListenFirehose(evtbusTestChannelName)
    responseHandler.Handle(
        func(msg *Message) {},
        func(err error) {})
    err := responseHandler.Fire()
    assert.Errorf(t, err, "nothing to fire, request is empty")
    destroyTestChannel()
}

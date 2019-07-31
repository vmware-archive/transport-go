// Copyright 2019 VMware Inc.
package bus

import (
    "errors"
    "github.com/stretchr/testify/assert"
    "testing"
)

var evtBusTest EventBus
var evtbusTestChannelName string = "test-channel"
var evtbusTestManager ChannelManager

func init() {
    evtBusTest = GetBus()
}

func createTestChannel() {
    evtbusTestManager = evtBusTest.GetChannelManager()
    evtbusTestManager.CreateChannel(evtbusTestChannelName)
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
    err := evtBusTest.SendResponseMessage("channel-not-here", "hello melody")
    assert.NotNil(t, err)
}

func TestEventBus_SendRequestMessageNoChannel(t *testing.T) {
    err := evtBusTest.SendRequestMessage("channel-not-here", "hello melody")
    assert.NotNil(t, err)
}

func TestEventBus_ListenStream(t *testing.T) {
    createTestChannel()
    handler, err := evtBusTest.ListenStream(evtbusTestChannelName)
    assert.Nil(t, err)
    assert.NotNil(t, handler)
    count := 0
    handler.handle(
        func(msg *Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 500; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "hello melody")

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1)
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
    handler.handle(
        func(msg *Message) {
            count++
        },
        func(err error) {})

    for i := 0; i < 300; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 0)

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1)
    }
    assert.Equal(t,1, count)
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
    handler.handle(
        func(msg *Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 300; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody")

        // send responses to make sure we're only getting requests
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion if picked up")
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion again")
    }
    assert.Equal(t, 300, count)
    destroyTestChannel()
}

func TestEventBus_ListenRequestStreamNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenRequestStream("missing-channel")
    assert.NotNil(t, err)
}


func TestEventBus_TestErrorMessageHandling(t *testing.T) {
    createTestChannel()

    err := evtBusTest.SendErrorMessage("invalid-channel", errors.New("something went wrong"))
    assert.NotNil(t, err)

    handler, _ := evtBusTest.ListenStream(evtbusTestChannelName)
    countSuccess, countError := 0, 0
    handler.handle(
        func(msg *Message) {
            countSuccess++
        },
        func(err error) {
            countError++
        })

    for i := 0; i < 300; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion if picked up")
        evtBusTest.SendErrorMessage(evtbusTestChannelName, errors.New("something went wrong"))
    }
    assert.Equal(t, 300, countSuccess)
    assert.Equal(t, 300, countError)
    destroyTestChannel()
}

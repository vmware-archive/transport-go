// Copyright 2019 VMware Inc.

package bus

import (
    "bifrost/model"
    "bifrost/util"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

var testChannelManager ChannelManager
var testChannelManagerChannelName string = "melody"

func createManager() ChannelManager {
    manager := NewBusChannelManager(createBus())
    return manager
}

func createBus() *bifrostEventBus {
    bf := new(bifrostEventBus)
    bf.init()
    return bf
}

func TestChannelManager_Boot(t *testing.T) {
    testChannelManager = createManager()
    assert.Len(t, testChannelManager.GetAllChannels(), 0)
}

func TestChannelManager_CreateChannel(t *testing.T) {
    testChannelManager = createManager()

    testChannelManager.CreateChannel(testChannelManagerChannelName)
    assert.Len(t, testChannelManager.GetAllChannels(), 1)

    fetchedChannel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.NotNil(t, fetchedChannel)
    assert.True(t, testChannelManager.CheckChannelExists(testChannelManagerChannelName))
}

func TestChannelManager_GetNotExistentChannel(t *testing.T) {
    testChannelManager = createManager()

    fetchedChannel, err := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.NotNil(t, err)
    assert.Nil(t, fetchedChannel)
}

func TestChannelManager_DestroyChannel(t *testing.T) {
    testChannelManager = createManager()

    testChannelManager.CreateChannel(testChannelManagerChannelName)
    testChannelManager.DestroyChannel(testChannelManagerChannelName)
    fetchedChannel, err := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, testChannelManager.GetAllChannels(), 0)
    assert.NotNil(t, err)
    assert.Nil(t, fetchedChannel)
}

func TestChannelManager_SubscribeChannelHandler(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    uuid, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.Nil(t, err)
    assert.NotNil(t, uuid)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
}

func TestChannelManager_SubscribeChannelHandlerMissingChannel(t *testing.T) {
    testChannelManager = createManager()
    handler := func(*model.Message) {}
    _, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandler(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    uuid, _ := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)

    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, uuid)
    assert.Nil(t, err)
    assert.Len(t, channel.eventHandlers, 0)
}

func TestChannelManager_UnsubscribeChannelHandlerMissingChannel(t *testing.T) {
    testChannelManager = createManager()
    uuid := uuid.New()
    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, &uuid)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandlerNoId(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
    id := uuid.New()
    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, &id)
    assert.NotNil(t, err)
    assert.Len(t, channel.eventHandlers, 1)
}

func TestChannelManager_TestWaitForGroupOnBadChannel(t *testing.T) {
    testChannelManager = createManager()
    err := testChannelManager.WaitForChannel("unknown")
    assert.Error(t, err, "no such Channel as 'unknown'")
}


func TestChannelManager_TestGalacticChannelOpen(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)
    d := make(chan bool)
    s := util.GetMonitor()
    var listenMonitor = func() {

        // mark channel as galactic.
        e := testChannelManager.MarkChannelAsGalactic(testChannelManagerChannelName, "/topic/testy-test")
        assert.Nil(t, e)

        for {
            e := <-s.Stream
            if e.EventType == util.ChannelIsGalacticEvt {
                assert.Equal(t, "/topic/testy-test", e.Message.Payload.(string))
                d <- true
                break
            }
        }
    }
    go listenMonitor()
    // wait until we get what we need.
    <-d
    util.ResetMonitor()
}

func TestChannelManager_TestGalacticChannelOpenError(t *testing.T) {
    // channel is not open / does not exist, so this should fail.
    e :=  testChannelManager.MarkChannelAsGalactic(evtbusTestChannelName, "/topic/testy-test")
    assert.Error(t, e)
    util.ResetMonitor()
}

func TestChannelManager_TestGalacticChannelCloseError(t *testing.T) {
    // channel is not open / does not exist, so this should fail.
    e := testChannelManager.MarkChannelAsLocal(evtbusTestChannelName)
    assert.Error(t, e)
    util.ResetMonitor()
}

func TestChannelManager_TestLocalChannel(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)
    d := make(chan bool)
    s := util.GetMonitor()
    var listenMonitor = func() {

        // map channel to galactic dest
        e := testChannelManager.MarkChannelAsLocal(testChannelManagerChannelName)
        assert.Nil(t, e)

        for {
            e := <-s.Stream
            if e.EventType == util.ChannelIsLocalEvt {
                d <- true
                break
            }
        }
    }

    go listenMonitor()
    // wait for us to get what we need.
    <-d
    util.ResetMonitor()
}

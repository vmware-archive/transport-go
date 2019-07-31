// Copyright 2019 VMware Inc.

package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

var testChannelManager ChannelManager
var testChannelManagerChannelName string = "melody"

func createManager() ChannelManager {
    manager := new(busChannelManager)
    manager.Boot()
    return manager
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

    handler := func(*Message) {}
    uuid, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.Nil(t, err)
    assert.NotNil(t, uuid)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
}

func TestChannelManager_SubscribeChannelHandlerMissingChannel(t *testing.T) {
    testChannelManager = createManager()
    handler := func(*Message) {}
    _, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandler(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*Message) {}
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

    handler := func(*Message) {}
    testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
    id := uuid.New()
    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, &id)
    assert.NotNil(t, err)
    assert.Len(t, channel.eventHandlers, 1)
}

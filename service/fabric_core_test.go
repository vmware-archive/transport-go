// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
    "testing"
    "go-bifrost/bus"
    "github.com/stretchr/testify/assert"
    "go-bifrost/model"
    "github.com/google/uuid"
    "sync"
)

func newTestFabricCore() FabricServiceCore {
    bus := bus.NewEventBusInstance()
    bus.GetChannelManager().CreateChannel("test-channel")
    return &fabricCore{
        channelName: "test-channel",
        bus: bus,
    }
}

func TestFabricCore_Bus(t *testing.T) {
    core := newTestFabricCore()
    assert.NotNil(t, core.Bus())
}

func TestFabricCore_SendMethods(t *testing.T) {
    core := newTestFabricCore()

    mh, _ := core.Bus().ListenStream("test-channel")

    wg := sync.WaitGroup{}

    var count = 0
    var lastMessage *model.Message

    mh.Handle(func(message *model.Message) {
        count++
        lastMessage = message
        wg.Done()
    }, func(e error) {
        assert.Fail(t, "unexpected error")
    })

    id := uuid.New()
    req := model.Request{
        Id: &id,
        Request: "test-request",
        BrokerDestination: &model.BrokerDestinationConfig{
            Destination: "test",
        },
    }

    wg.Add(1)
    core.SendResponse(&req, "test-response")
    wg.Wait()

    assert.Equal(t, count, 1)

    response, ok := lastMessage.Payload.(model.Response)
    assert.True(t, ok)
    assert.Equal(t, response.Id, req.Id)
    assert.Equal(t, response.Payload, "test-response")
    assert.False(t, response.Error)
    assert.Equal(t, response.BrokerDestination.Destination, "test")

    wg.Add(1)
    core.SendErrorResponse(&req, 404, "test-error")
    wg.Wait()

    assert.Equal(t, count, 2)
    response = lastMessage.Payload.(model.Response)

    assert.Equal(t, response.Id, req.Id)
    assert.Nil(t, response.Payload)
    assert.True(t, response.Error)
    assert.Equal(t, response.ErrorCode, 404)
    assert.Equal(t, response.ErrorMessage, "test-error")

    wg.Add(1)
    core.HandleUnknownRequest(&req)
    wg.Wait()

    assert.Equal(t, count, 3)
    response = lastMessage.Payload.(model.Response)

    assert.Equal(t, response.Id, req.Id)
    assert.Nil(t, response.Payload)
    assert.True(t, response.Error)
    assert.Equal(t, response.ErrorCode, 1)
    assert.Equal(t, response.ErrorMessage, "unsupported request: " + req.Request)
}
// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.eng.vmware.com/bifrost/go-bifrost/bus"
	"gitlab.eng.vmware.com/bifrost/go-bifrost/model"
	"sync"
	"testing"
)

func newTestFabricCore(channelName string) FabricServiceCore {
	eventBus := bus.NewEventBusInstance()
	eventBus.GetChannelManager().CreateChannel(channelName)
	return &fabricCore{
		channelName: channelName,
		bus:         eventBus,
	}
}

func TestFabricCore_Bus(t *testing.T) {
	core := newTestFabricCore("test-channel")
	assert.NotNil(t, core.Bus())
}

func TestFabricCore_SendMethods(t *testing.T) {
	core := newTestFabricCore("test-channel")

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
		Id:      &id,
		Request: "test-request",
		BrokerDestination: &model.BrokerDestinationConfig{
			Destination: "test",
		},
	}

	wg.Add(1)
	core.SendResponse(&req, "test-response")
	wg.Wait()

	assert.Equal(t, count, 1)

	response, ok := lastMessage.Payload.(*model.Response)
	assert.True(t, ok)
	assert.Equal(t, response.Id, req.Id)
	assert.Equal(t, response.Payload, "test-response")
	assert.False(t, response.Error)
	assert.Equal(t, response.BrokerDestination.Destination, "test")

	wg.Add(1)
	h := make(map[string]string)
	h["hello"] = "there"
	core.SendResponseWithHeaders(&req, "test-response-with-headers", h)
	wg.Wait()

	assert.Equal(t, count, 2)

	response, ok = lastMessage.Payload.(*model.Response)
	assert.True(t, ok)
	assert.Equal(t, response.Id, req.Id)
	assert.Equal(t, response.Payload, "test-response-with-headers")
	assert.False(t, response.Error)
	assert.Equal(t, response.BrokerDestination.Destination, "test")
	assert.Equal(t, response.Headers["hello"], "there")

	wg.Add(1)
	core.SendErrorResponse(&req, 404, "test-error")
	wg.Wait()

	assert.Equal(t, count, 3)
	response = lastMessage.Payload.(*model.Response)

	assert.Equal(t, response.Id, req.Id)
	assert.Nil(t, response.Payload)
	assert.True(t, response.Error)
	assert.Equal(t, response.ErrorCode, 404)
	assert.Equal(t, response.ErrorMessage, "test-error")

	wg.Add(1)
	core.HandleUnknownRequest(&req)
	wg.Wait()

	assert.Equal(t, count, 4)
	response = lastMessage.Payload.(*model.Response)

	assert.Equal(t, response.Id, req.Id)
	assert.False(t, response.Error)
	assert.Equal(t, response.ErrorCode, 0)
	assert.Equal(t, response.Payload, "unsupported request for \"test-channel\": "+req.Request)
}

func TestFabricCore_RestServiceRequest(t *testing.T) {

	core := newTestFabricCore("test-channel")

	core.Bus().GetChannelManager().CreateChannel(restServiceChannel)

	var lastRequest *model.Request

	wg := sync.WaitGroup{}

	mh, _ := core.Bus().ListenRequestStream(restServiceChannel)
	mh.Handle(
		func(message *model.Message) {
			lastRequest = message.Payload.(*model.Request)
			wg.Done()
		},
		func(e error) {})

	var lastSuccess, lastError *model.Response

	restRequest := &RestServiceRequest{
		Uri:     "test",
		Headers: map[string]string{"h1": "value1"},
	}

	wg.Add(1)
	core.RestServiceRequest(restRequest, func(response *model.Response) {
		lastSuccess = response
		wg.Done()
	}, func(response *model.Response) {
		lastError = response
		wg.Done()
	})

	wg.Wait()

	wg.Add(1)
	core.Bus().SendResponseMessage(restServiceChannel, &model.Response{
		Payload: "test",
	}, lastRequest.Id)
	wg.Wait()

	assert.NotNil(t, lastSuccess)
	assert.Nil(t, lastError)

	assert.Equal(t, lastRequest.Payload, restRequest)
	assert.Equal(t, len(lastRequest.Payload.(*RestServiceRequest).Headers), 1)
	assert.Equal(t, lastRequest.Payload.(*RestServiceRequest).Headers["h1"], "value1")
	assert.Equal(t, lastSuccess.Payload, "test")

	lastSuccess, lastError = nil, nil

	core.SetHeaders(map[string]string{"h2": "value2", "h1": "new-value"})

	wg.Add(1)
	core.RestServiceRequest(restRequest, func(response *model.Response) {
		lastSuccess = response
		wg.Done()
	}, func(response *model.Response) {
		lastError = response
		wg.Done()
	})

	wg.Wait()

	wg.Add(1)
	core.Bus().SendResponseMessage(restServiceChannel, &model.Response{
		ErrorMessage: "error",
		Error:        true,
		ErrorCode:    1,
	}, lastRequest.Id)
	wg.Wait()

	assert.Nil(t, lastSuccess)
	assert.NotNil(t, lastError)
	assert.Equal(t, lastError.ErrorMessage, "error")
	assert.Equal(t, lastError.ErrorCode, 1)

	assert.Equal(t, len(lastRequest.Payload.(*RestServiceRequest).Headers), 2)
	assert.Equal(t, lastRequest.Payload.(*RestServiceRequest).Headers["h1"], "value1")
	assert.Equal(t, lastRequest.Payload.(*RestServiceRequest).Headers["h2"], "value2")

	lastSuccess, lastError = nil, nil
	wg.Add(1)
	core.RestServiceRequest(restRequest, func(response *model.Response) {
		lastSuccess = response
		wg.Done()
	}, func(response *model.Response) {
		lastError = response
		wg.Done()
	})

	wg.Wait()

	wg.Add(1)
	core.Bus().SendErrorMessage(restServiceChannel, errors.New("test-error"), lastRequest.Id)
	wg.Wait()

	assert.Nil(t, lastSuccess)
	assert.NotNil(t, lastError)
	assert.Equal(t, lastError.ErrorMessage, "test-error")
	assert.Equal(t, lastError.ErrorCode, 500)
}

// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
	"fmt"
	"github.com/google/uuid"
	"gitlab.eng.vmware.com/bifrost/go-bifrost/bus"
	"gitlab.eng.vmware.com/bifrost/go-bifrost/model"
)

// Interface providing base functionality to fabric services.
type FabricServiceCore interface {
	// Returns the EventBus instance.
	Bus() bus.EventBus
	// Uses the "responsePayload" and "request" params to build and send model.Response object
	// on the service channel.
	SendResponse(request *model.Request, responsePayload interface{})
	// Same as SendResponse, but include headers. Useful for HTTP REST interfaces - these headers will be
	// set as HTTP response headers. Great for custom mime-types, binary stuff and more.
	SendResponseWithHeaders(request *model.Request, responsePayload interface{}, headers map[string]string)
	// Builds an error model.Response object and sends it on the service channel as
	// response to the "request" param.
	SendErrorResponse(request *model.Request, responseErrorCode int, responseErrorMessage string)
	SendErrorResponseWithPayload(request *model.Request, responseErrorCode int, responseErrorMessage string, payload interface{})
	// Handles unknown/unsupported request.
	HandleUnknownRequest(request *model.Request)
	// Make a new RestService call.
	RestServiceRequest(restRequest *RestServiceRequest,
		successHandler model.ResponseHandlerFunction, errorHandler model.ResponseHandlerFunction)
	// Set global headers for a given fabric service (each service has its own set of global headers).
	// The headers will be applied to all requests made by this instance's RestServiceRequest method.
	// Global header values can be overridden per request via the RestServiceRequest.Headers property.
	SetHeaders(headers map[string]string)
}

type fabricCore struct {
	channelName string
	bus         bus.EventBus
	headers     map[string]string
}

func (core *fabricCore) Bus() bus.EventBus {
	return core.bus
}

func (core *fabricCore) SendResponse(request *model.Request, responsePayload interface{}) {
	response := &model.Response{
		Id:                request.Id,
		Destination:       core.channelName,
		Payload:           responsePayload,
		BrokerDestination: request.BrokerDestination,
	}
	core.bus.SendResponseMessage(core.channelName, response, request.Id)
}

func (core *fabricCore) SendResponseWithHeaders(request *model.Request, responsePayload interface{}, headers map[string]string) {
	response := &model.Response{
		Id:                request.Id,
		Destination:       core.channelName,
		Payload:           responsePayload,
		BrokerDestination: request.BrokerDestination,
		Headers:           headers,
	}
	core.bus.SendResponseMessage(core.channelName, response, request.Id)
}

func (core *fabricCore) SendErrorResponse(
	request *model.Request, responseErrorCode int, responseErrorMessage string) {
	core.SendErrorResponseWithPayload(request, responseErrorCode, responseErrorMessage, nil)
}

func (core *fabricCore) SendErrorResponseWithPayload(
	request *model.Request,
	responseErrorCode int, responseErrorMessage string, payload interface{}) {

	response := &model.Response{
		Id:                request.Id,
		Destination:       core.channelName,
		Payload:           payload,
		Error:             true,
		ErrorCode:         responseErrorCode,
		ErrorMessage:      responseErrorMessage,
		BrokerDestination: request.BrokerDestination,
	}
	core.bus.SendResponseMessage(core.channelName, response, request.Id)
}

func (core *fabricCore) HandleUnknownRequest(request *model.Request) {
	errorMsg := fmt.Sprintf("unsupported request for \"%s\": %s", core.channelName, request.Request)
	core.SendErrorResponse(request, 403, errorMsg)
}

func (core *fabricCore) SetHeaders(headers map[string]string) {
	core.headers = headers
}

func (core *fabricCore) RestServiceRequest(restRequest *RestServiceRequest,
	successHandler model.ResponseHandlerFunction, errorHandler model.ResponseHandlerFunction) {

	// merge global service headers with the headers from the httpRequest
	// note that headers specified in restRequest override the global headers.
	mergedHeaders := make(map[string]string)
	for k, v := range core.headers {
		mergedHeaders[k] = v
	}
	for k, v := range restRequest.Headers {
		mergedHeaders[k] = v
	}
	restRequest.Headers = mergedHeaders

	id := uuid.New()
	request := &model.Request{
		Id:      &id,
		Payload: restRequest,
	}
	mh, _ := core.bus.ListenOnceForDestination(restServiceChannel, request.Id)
	mh.Handle(func(message *model.Message) {
		response := message.Payload.(*model.Response)
		if response.Error {
			errorHandler(response)
		} else {
			successHandler(response)
		}
	}, func(e error) {
		errorHandler(&model.Response{
			Error:        true,
			ErrorMessage: e.Error(),
			ErrorCode:    500,
		})
	})
	core.bus.SendRequestMessage(restServiceChannel, request, request.Id)
}

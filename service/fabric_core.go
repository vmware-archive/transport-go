// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
    "go-bifrost/bus"
    "go-bifrost/model"
    "fmt"
)

// Interface providing base functionality to fabric services.
type FabricServiceCore interface {
    // Returns the EventBus instance.
    Bus() bus.EventBus
    // Uses the "responsePayload" and "request" params to build and send model.Response object
    // on the service channel.
    SendResponse(request *model.Request, responsePayload interface{})
    // Builds an error model.Response object and sends it on the service channel as
    // response to the "request" param.
    SendErrorResponse(request *model.Request, responseErrorCode int, responseErrorMessage string)
    // Handles unknown/unsupported request.
    HandleUnknownRequest(request *model.Request)
}

type fabricCore struct {
    channelName string
    bus         bus.EventBus
}

func (core *fabricCore) Bus() bus.EventBus {
    return core.bus
}

func (core *fabricCore) SendResponse(request *model.Request, responsePayload interface{}) {
    response := model.Response{
        Id:          request.Id,
        Destination: core.channelName,
        Payload:     responsePayload,
    }
    core.bus.SendResponseMessage(core.channelName, response, request.Id)
}

func (core *fabricCore) SendErrorResponse(
        request *model.Request, responseErrorCode int, responseErrorMessage string) {

    response := model.Response{
        Id:           request.Id,
        Destination:  core.channelName,
        Error:        true,
        ErrorCode:    responseErrorCode,
        ErrorMessage: responseErrorMessage,
    }
    core.bus.SendResponseMessage(core.channelName, response, request.Id)
}

func (core *fabricCore) HandleUnknownRequest(request *model.Request) {
    core.SendErrorResponse(request, 1, fmt.Sprintf("unsupported request: %s", request.Request))
}
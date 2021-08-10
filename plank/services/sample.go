// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package services

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/service"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	PingPongServiceChan = "ping-pong-service"
)

type PingPongService struct{}

func NewPingPongService() *PingPongService {
	return &PingPongService{}
}

func (ps *PingPongService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	switch request.Request {
	case "ping":
		m := make(map[string]interface{})
		m["timestamp"] = time.Now().Unix()
		err := json.Unmarshal(request.Payload.([]byte), &m)
		if err != nil {
			core.SendErrorResponse(request, 400, err.Error())
		} else {
			core.SendResponse(request, m)
		}
	case "ping2":
		rsp := make(map[string]interface{})
		val := request.Payload.(string)
		rsp["payload"] = val + "-response"
		rsp["timestamp"] = time.Now().Unix()
		core.SendResponse(request, rsp)
	default:
		core.HandleUnknownRequest(request)
	}
}

func (ps *PingPongService) OnServiceReady() chan bool {
	// for sample purposes this service initializes instantly
	readyChan := make(chan bool, 1)
	readyChan <- true
	return readyChan
}

func (ps *PingPongService) OnServerShutdown() {
	// for sample purposes emulate a 1 second teardown process
	time.Sleep(1 * time.Second)
}

func (ps *PingPongService) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return []*service.RESTBridgeConfig{
		{
			ServiceChannel: PingPongServiceChan,
			Uri:            "/rest/ping-pong",
			Method:         http.MethodPost,
			AllowHead:      true,
			AllowOptions:   true,
			FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
				body, _ := ioutil.ReadAll(r.Body)
				return createServiceRequest("ping", body)
			},
		},
		{
			ServiceChannel: PingPongServiceChan,
			Uri:            "/rest/ping-pong2",
			Method:         http.MethodGet,
			AllowHead:      true,
			AllowOptions:   true,
			FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
				return model.Request{Id: &uuid.UUID{}, Request: "ping2", Payload: r.URL.Query().Get("message")}
			},
		},
		{
			ServiceChannel: PingPongServiceChan,
			Uri:            "/rest/ping-pong/{from}/{to}/{message}",
			Method:         http.MethodGet,
			FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
				pathParams := mux.Vars(r)
				return model.Request{
					Id:      &uuid.UUID{},
					Request: "ping2",
					Payload: fmt.Sprintf(
						"From %s to %s: %s",
						pathParams["from"],
						pathParams["to"],
						pathParams["message"])}
			},
		},
	}
}

func createServiceRequest(requestType string, body []byte) model.Request {
	id := uuid.New()
	return model.Request{
		Id:      &id,
		Request: requestType,
		Payload: body}
}

func createServiceRequestWithValues(requestType string, vals url.Values) model.Request {
	id := uuid.New()
	return model.Request{
		Id:      &id,
		Request: requestType,
		Payload: vals}
}

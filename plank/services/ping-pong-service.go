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
	"time"
)

const (
	PingPongServiceChan = "ping-pong-service"
)

// PingPongService is a very simple service to demonstrate how request-response cycles are handled in Transport & Plank.
// this service has two requests named "ping-post" and "ping-get", the first accepts the payload and expects it to be of
// a POJO type (e.g. {"anything": "here"}), whereas the second expects the payload to be a pure string.
// a request made through the Event Bus API like bus.RequestOnce() will be routed to HandleServiceRequest()
// which will match the request's Request to the list of available service request types and return the response.
type PingPongService struct{}

func NewPingPongService() *PingPongService {
	return &PingPongService{}
}

// Init will fire when the service is being registered by the fabric, it passes a reference of the same core
// Passed through when implementing HandleServiceRequest
func (ps *PingPongService) Init(core service.FabricServiceCore) error {

	// set default headers for this service.
	core.SetHeaders(map[string]string{
		"Content-Type": "application/json",
	})

	return nil
}

// HandleServiceRequest routes the incoming request and based on the Request property of request, it invokes the
// appropriate handler logic defined and separated by a switch statement like the one shown below.
func (ps *PingPongService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	switch request.Request {
	// ping-post request type accepts the payload as a POJO
	case "ping-post":
		m := make(map[string]interface{})
		m["timestamp"] = time.Now().Unix()
		err := json.Unmarshal(request.Payload.([]byte), &m)
		if err != nil {
			core.SendErrorResponse(request, 400, err.Error())
		} else {
			core.SendResponse(request, m)
		}
	// ping-get request type accepts the payload as a string
	case "ping-get":
		rsp := make(map[string]interface{})
		val := request.Payload.(string)
		rsp["payload"] = val + "-response"
		rsp["timestamp"] = time.Now().Unix()
		core.SendResponse(request, rsp)
	default:
		core.HandleUnknownRequest(request)
	}
}

// OnServiceReady contains logic that handles the service initialization that needs to be carried out
// before it is ready to accept user requests. Plank monitors and waits for service initialization to
// complete by trying to receive a boolean payload from a channel of boolean type. as a service developer
// you need to perform any and every init logic here and return a channel that would receive a payload
// once your service truly becomes ready to accept requests.
func (ps *PingPongService) OnServiceReady() chan bool {
	// for sample purposes this service initializes instantly
	readyChan := make(chan bool, 1)
	readyChan <- true
	return readyChan
}

// OnServerShutdown is the opposite of OnServiceReady. it is called when the server enters graceful shutdown
// where all the running services need to complete before the server could shut down finally. this method does not need
// to return anything because the main server thread is going to shut down soon, but if there's any important teardown
// or cleanup that needs to be done, this is the right place to perform that.
func (ps *PingPongService) OnServerShutdown() {
	// for sample purposes emulate a 1 second teardown process
	time.Sleep(1 * time.Second)
}

// GetRESTBridgeConfig returns a list of REST bridge configurations that Plank will use to automatically register
// REST endpoints that map to the requests for this service. this means you can map any request types defined under
// HandleServiceRequest with any combination of URI, HTTP verb, path parameter, query parameter and request headers.
// as the service author you have full control over every aspect of the translation process which basically turns
// an incoming *http.Request into model.Request. See FabricRequestBuilder below to see it in action.
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
				return model.CreateServiceRequest("ping-post", body)
			},
		},
		{
			ServiceChannel: PingPongServiceChan,
			Uri:            "/rest/ping-pong2",
			Method:         http.MethodGet,
			AllowHead:      true,
			AllowOptions:   true,
			FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
				return model.Request{Id: &uuid.UUID{}, Request: "ping-get", Payload: r.URL.Query().Get("message")}
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
					Request: "ping-get",
					Payload: fmt.Sprintf(
						"From %s to %s: %s",
						pathParams["from"],
						pathParams["to"],
						pathParams["message"])}
			},
		},
	}
}

// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package services

import (
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/service"
	"time"
)

const (
	Sample2ServiceChan = "sample-2-service"
)

type Sample2Service struct{}

func NewSample2Service() *Sample2Service {
	return &Sample2Service{}
}

func (ps *Sample2Service) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	switch request.Request {
	case "echo":
		rsp := make(map[string]interface{})
		val := request.Payload.(string)
		rsp["payload"] = val + "from server: "
		rsp["timestamp"] = time.Now().Unix()
		core.SendResponse(request, rsp)
	default:
		core.HandleUnknownRequest(request)
	}
}

func (ps *Sample2Service) OnServiceReady() chan bool {
	// for sample purposes this service emulates a 5 second initialization time
	readyChan := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		readyChan <- true
	}()
	return readyChan
}

func (ps *Sample2Service) OnServerShutdown() {
	// for sample purposes emulate a 5 second teardown process
	time.Sleep(5 * time.Second)
}

func (ps *Sample2Service) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return nil
}

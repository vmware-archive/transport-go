// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/plank/pkg/server"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/plank/utils"
	"os"
	"time"
)

// configure flags
func main() {
	serverConfig, err := server.CreateServerConfig()
	if err != nil {
		utils.Log.Fatalln(err)
	}
	platformServer := server.NewPlatformServer(serverConfig)
	brokerConfigForAnotherWs := &bridge.BrokerConnectorConfig{
		Username:        "guest",
		Password:        "guest",
		ServerAddr:      "localhost:30081",
		UseWS:           true,
		WebSocketConfig: &bridge.WebSocketConfig{WSPath: "/ws"},
		HeartBeatOut:    30 * time.Second,
		STOMPHeader:     map[string]string{},
	}
	if err = platformServer.RegisterService(services.NewExternalBrokerExampleService(bus.GetBus(), brokerConfigForAnotherWs), services.ExternalBrokerExampleServiceChannel); err != nil {
		utils.Log.Fatalln(err)
	}

	syschan := make(chan os.Signal, 1)
	platformServer.StartServer(syschan)
}

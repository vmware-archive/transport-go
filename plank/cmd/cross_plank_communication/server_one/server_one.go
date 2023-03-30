// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"github.com/vmware/transport-go/plank/pkg/server"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/plank/utils"
	"os"
)

// configure flags
func main() {
	serverConfig, err := server.CreateServerConfig()
	if err != nil {
		utils.Log.Fatalln(err)
	}
	platformServer := server.NewPlatformServer(serverConfig)
	if err = platformServer.RegisterService(services.NewPingPongService(), services.PingPongServiceChan); err != nil {
		utils.Log.Fatalln(err)
	}

	syschan := make(chan os.Signal, 1)
	platformServer.StartServer(syschan)
}

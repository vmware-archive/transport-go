package main

import (
	"github.com/jooskim/plank/pkg/server"
	"github.com/jooskim/plank/services"
	"github.com/jooskim/plank/utils"
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

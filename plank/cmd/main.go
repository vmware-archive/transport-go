package main

import (
	"github.com/vmware/transport-go/plank/pkg/server"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/urfave/cli"
	"os"
)

var version string
var platformServer *server.PlatformServer

func main() {
	app := cli.NewApp()
	app.Name = "Plank"
	app.Version = version
	app.Description = "Plank"
	app.Commands = []cli.Command{
		{
			Name:  "start-server",
			Usage: "Start server",
			Flags: utils.UrfaveCLIFlags,
			Action: func(c *cli.Context) error {
				var platformServer server.PlatformServer
				var err error
				configPath := c.String("config-file")

				// if server config file is available create a server instance with it. otherwise, create it with a
				// server config with provided CLI flag values
				if len(configPath) > 0 {
					platformServer, err = server.NewPlatformServerFromConfig(configPath)
					if err != nil {
						return err
					}
				} else {
					serverConfig, err := server.CreateServerConfigFromUrfaveCLIContext(c)
					if err != nil {
						return err
					}
					platformServer = server.NewPlatformServer(serverConfig)
				}

				// register services
				if err := platformServer.RegisterService(services.NewPingPongService(), services.PingPongServiceChan); err != nil {
					panic(err)
				}
				if err := platformServer.RegisterService(services.NewSample2Service(), services.Sample2ServiceChan); err != nil {
					panic(err)
				}

				// start server
				syschan := make(chan os.Signal, 1)
				platformServer.StartServer(syschan)

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

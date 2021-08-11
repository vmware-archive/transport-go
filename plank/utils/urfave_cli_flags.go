// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"fmt"
	"github.com/urfave/cli"
)

var UrfaveCLIFlags = []cli.Flag{
	cli.StringFlag{
		Name:   fmt.Sprintf("%s, %s", PlatformServerFlagConstants["Hostname"]["FlagName"], "n"),
		EnvVar: "PLANK_HOSTNAME",
		Value:  "localhost",
		Usage:  PlatformServerFlagConstants["Hostname"]["Description"],
	},
	cli.IntFlag{
		Name:   fmt.Sprintf("%s, %s", PlatformServerFlagConstants["Port"]["FlagName"], "p"),
		Usage:  PlatformServerFlagConstants["Port"]["Description"],
		EnvVar: "PLANK_PORT",
		Value:  30080,
	},
	cli.StringFlag{
		Name:   fmt.Sprintf("%s, %s", PlatformServerFlagConstants["RootDir"]["FlagName"], "r"),
		Usage:  PlatformServerFlagConstants["RootDir"]["Description"],
		EnvVar: "PLANK_ROOT",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["Cert"]["FlagName"]),
		Usage: PlatformServerFlagConstants["Cert"]["Description"],
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["CertKey"]["FlagName"]),
		Usage: PlatformServerFlagConstants["CertKey"]["Description"],
	},
	cli.StringSliceFlag{
		Name:  fmt.Sprintf("%s, %s", PlatformServerFlagConstants["Static"]["FlagName"], "s"),
		Usage: PlatformServerFlagConstants["Static"]["Description"],
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["SpaPath"]["FlagName"]),
		Usage: PlatformServerFlagConstants["SpaPath"]["Description"],
	},
	cli.BoolFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["NoFabricBroker"]["FlagName"]),
		Usage: PlatformServerFlagConstants["NoFabricBroker"]["Description"],
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["FabricEndpoint"]["FlagName"]),
		Usage: PlatformServerFlagConstants["FabricEndpoint"]["Description"],
		Value: "/ws",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["TopicPrefix"]["FlagName"]),
		Usage: PlatformServerFlagConstants["TopicPrefix"]["Description"],
		Value: "/topic",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["QueuePrefix"]["FlagName"]),
		Usage: PlatformServerFlagConstants["QueuePrefix"]["Description"],
		Value: "/queue",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["RequestPrefix"]["FlagName"]),
		Usage: PlatformServerFlagConstants["RequestPrefix"]["Description"],
		Value: "/pub",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["RequestQueuePrefix"]["FlagName"]),
		Usage: PlatformServerFlagConstants["RequestQueuePrefix"]["Description"],
		Value: "/pub/queue",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["ConfigFile"]["FlagName"]),
		Usage: PlatformServerFlagConstants["ConfigFile"]["Description"],
	},
	cli.Int64Flag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["ShutdownTimeout"]["FlagName"]),
		Usage: PlatformServerFlagConstants["ShutdownTimeout"]["Description"],
		Value: 5,
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s, %s", PlatformServerFlagConstants["OutputLog"]["FlagName"], "l"),
		Usage: PlatformServerFlagConstants["OutputLog"]["Description"],
		Value: "stdout",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s, %s", PlatformServerFlagConstants["AccessLog"]["FlagName"], "a"),
		Usage: PlatformServerFlagConstants["AccessLog"]["Description"],
		Value: "stdout",
	},
	cli.StringFlag{
		Name:  fmt.Sprintf("%s, %s", PlatformServerFlagConstants["ErrorLog"]["FlagName"], "e"),
		Usage: PlatformServerFlagConstants["ErrorLog"]["Description"],
		Value: "stderr",
	},
	cli.BoolFlag{
		Name:  fmt.Sprintf("%s, %s", PlatformServerFlagConstants["Debug"]["FlagName"], "d"),
		Usage: PlatformServerFlagConstants["Debug"]["Description"],
	},
	cli.BoolFlag{
		Name:  fmt.Sprintf("%s, %s", PlatformServerFlagConstants["NoBanner"]["FlagName"], "b"),
		Usage: PlatformServerFlagConstants["NoBanner"]["Description"],
	},
	cli.BoolFlag{
		Name:  fmt.Sprintf("%s", PlatformServerFlagConstants["Prometheus"]["FlagName"]),
		Usage: PlatformServerFlagConstants["Prometheus"]["Description"],
	},
	cli.Int64Flag{
		Name: fmt.Sprintf("%s", PlatformServerFlagConstants["RestBridgeTimeout"]["FlagName"]),
		Usage: PlatformServerFlagConstants["RestBridgeTimeout"]["Description"],
		Value: 1,
	},
}

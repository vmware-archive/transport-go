// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/vmware/transport-go/plank/utils"
	"os"
)

type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	return fmt.Sprint(*f)
}

func (f *stringSliceFlag) Type() string {
	return "stringSlice"
}

func (f *stringSliceFlag) Set(value string) error {
	if f == nil {
		*f = make([]string, 0)
	}
	*f = append(*f, value)
	return nil
}

type serverConfigFactory struct {
	staticPtr stringSliceFlag
	flagSet   *flag.FlagSet
	flagsParsed bool
}

func (f *serverConfigFactory) Hostname() string {
	return viper.GetString(utils.PlatformServerFlagConstants["Hostname"]["FlagName"])
}

func (f *serverConfigFactory) Port() int {
	return viper.GetInt(utils.PlatformServerFlagConstants["Port"]["FlagName"])
}

func (f *serverConfigFactory) RootDir() string {
	return viper.GetString(utils.PlatformServerFlagConstants["RootDir"]["FlagName"])
}

func (f *serverConfigFactory) Cert() string {
	return viper.GetString(utils.PlatformServerFlagConstants["Cert"]["FlagName"])
}

func (f *serverConfigFactory) CertKey() string {
	return viper.GetString(utils.PlatformServerFlagConstants["CertKey"]["FlagName"])
}

func (f *serverConfigFactory) Static() []string {
	return viper.GetStringSlice(utils.PlatformServerFlagConstants["Static"]["FlagName"])
}

func (f *serverConfigFactory) SpaPath() string {
	return viper.GetString(utils.PlatformServerFlagConstants["SpaPath"]["FlagName"])
}

func (f *serverConfigFactory) NoFabricBroker() bool {
	return viper.GetBool(utils.PlatformServerFlagConstants["NoFabricBroker"]["FlagName"])
}

func (f *serverConfigFactory) FabricEndpoint() string {
	return viper.GetString(utils.PlatformServerFlagConstants["FabricEndpoint"]["FlagName"])
}

func (f *serverConfigFactory) TopicPrefix() string {
	return viper.GetString(utils.PlatformServerFlagConstants["TopicPrefix"]["FlagName"])
}

func (f *serverConfigFactory) QueuePrefix() string {
	return viper.GetString(utils.PlatformServerFlagConstants["QueuePrefix"]["FlagName"])
}

func (f *serverConfigFactory) RequestPrefix() string {
	return viper.GetString(utils.PlatformServerFlagConstants["RequestPrefix"]["FlagName"])
}

func (f *serverConfigFactory) RequestQueuePrefix() string {
	return viper.GetString(utils.PlatformServerFlagConstants["RequestQueuePrefix"]["FlagName"])
}

func (f *serverConfigFactory) ConfigFile() string {
	return viper.GetString(utils.PlatformServerFlagConstants["ConfigFile"]["FlagName"])
}

func (f *serverConfigFactory) ShutdownTimeout() int64 {
	return viper.GetInt64(utils.PlatformServerFlagConstants["ShutdownTimeout"]["FlagName"])
}

func (f *serverConfigFactory) OutputLog() string {
	return viper.GetString(utils.PlatformServerFlagConstants["OutputLog"]["FlagName"])
}

func (f *serverConfigFactory) AccessLog() string {
	return viper.GetString(utils.PlatformServerFlagConstants["AccessLog"]["FlagName"])
}

func (f *serverConfigFactory) ErrorLog() string {
	return viper.GetString(utils.PlatformServerFlagConstants["ErrorLog"]["FlagName"])
}

func (f *serverConfigFactory) Debug() bool {
	return viper.GetBool(utils.PlatformServerFlagConstants["Debug"]["FlagName"])
}

func (f *serverConfigFactory) NoBanner() bool {
	return viper.GetBool(utils.PlatformServerFlagConstants["NoBanner"]["FlagName"])
}

func (f *serverConfigFactory) Prometheus() bool {
	return viper.GetBool(utils.PlatformServerFlagConstants["Prometheus"]["FlagName"])
}

func (f *serverConfigFactory) RestBridgeTimeout() int64 {
	return viper.GetInt64(utils.PlatformServerFlagConstants["RestBridgeTimeout"]["FlagName"])
}

// parseFlags reads OS arguments into the FlagSet in this factory instance
func (f *serverConfigFactory) parseFlags() {
	f.flagSet.Parse(os.Args[1:])
	f.flagsParsed = flag.Parsed()
}

// configureFlags defines flags definitions as well as associate environment variables for a few
// flags. see configureFlagsInFlagSet() for detailed flag defining logic.
func (f *serverConfigFactory) configureFlags(flagset *flag.FlagSet) {
	viper.SetEnvPrefix("PLANK_SERVER")
	viper.BindEnv("hostname")
	viper.BindEnv("port")
	viper.BindEnv("rootdir")
	f.flagSet = flagset
	f.configureFlagsInFlagSet(f.flagSet)
	viper.BindPFlags(f.flagSet)
}

// configureFlagsInFlagSet takes the pointer to an arbitrary FlagSet instance and
// populates it with the flag definitions necessary for PlatformServerConfig.
func (f *serverConfigFactory) configureFlagsInFlagSet(fs *flag.FlagSet) {
	wd, _ := os.Getwd()
	fs.StringP(
		utils.PlatformServerFlagConstants["Hostname"]["FlagName"],
		utils.PlatformServerFlagConstants["Hostname"]["ShortFlag"],
		"localhost",
		utils.PlatformServerFlagConstants["Hostname"]["Description"])
	fs.IntP(
		utils.PlatformServerFlagConstants["Port"]["FlagName"],
		utils.PlatformServerFlagConstants["Port"]["ShortFlag"],
		30080,
		utils.PlatformServerFlagConstants["Port"]["Description"])
	fs.StringP(
		utils.PlatformServerFlagConstants["RootDir"]["FlagName"],
		utils.PlatformServerFlagConstants["RootDir"]["ShortFlag"],
		wd,
		utils.PlatformServerFlagConstants["RootDir"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["Cert"]["FlagName"],
		"",
		utils.PlatformServerFlagConstants["Cert"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["CertKey"]["FlagName"],
		"",
		utils.PlatformServerFlagConstants["CertKey"]["Description"])
	fs.VarP(
		&f.staticPtr,
		utils.PlatformServerFlagConstants["Static"]["FlagName"],
		utils.PlatformServerFlagConstants["Static"]["ShortFlag"],
		utils.PlatformServerFlagConstants["Static"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["SpaPath"]["FlagName"],
		"",
		utils.PlatformServerFlagConstants["SpaPath"]["Description"])
	fs.Bool(
		utils.PlatformServerFlagConstants["NoFabricBroker"]["FlagName"],
		false,
		utils.PlatformServerFlagConstants["NoFabricBroker"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["FabricEndpoint"]["FlagName"],
		"/ws",
		utils.PlatformServerFlagConstants["FabricEndpoint"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["TopicPrefix"]["FlagName"],
		"/topic",
		utils.PlatformServerFlagConstants["TopicPrefix"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["QueuePrefix"]["FlagName"],
		"/queue",
		utils.PlatformServerFlagConstants["QueuePrefix"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["RequestPrefix"]["FlagName"],
		"/pub",
		utils.PlatformServerFlagConstants["RequestPrefix"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["RequestQueuePrefix"]["FlagName"],
		"/pub/queue",
		utils.PlatformServerFlagConstants["RequestQueuePrefix"]["Description"])
	fs.String(
		utils.PlatformServerFlagConstants["ConfigFile"]["FlagName"],
		"",
		utils.PlatformServerFlagConstants["ConfigFile"]["Description"])
	fs.Int64(
		utils.PlatformServerFlagConstants["ShutdownTimeout"]["FlagName"],
		5,
		utils.PlatformServerFlagConstants["ShutdownTimeout"]["Description"])
	fs.StringP(
		utils.PlatformServerFlagConstants["OutputLog"]["FlagName"],
		utils.PlatformServerFlagConstants["OutputLog"]["ShortFlag"],
		"stdout",
		utils.PlatformServerFlagConstants["OutputLog"]["Description"])
	fs.StringP(
		utils.PlatformServerFlagConstants["AccessLog"]["FlagName"],
		utils.PlatformServerFlagConstants["AccessLog"]["ShortFlag"],
		"stdout",
		utils.PlatformServerFlagConstants["AccessLog"]["Description"])
	fs.StringP(
		utils.PlatformServerFlagConstants["ErrorLog"]["FlagName"],
		utils.PlatformServerFlagConstants["ErrorLog"]["ShortFlag"],
		"stderr",
		utils.PlatformServerFlagConstants["ErrorLog"]["Description"])
	fs.BoolP(
		utils.PlatformServerFlagConstants["Debug"]["FlagName"],
		utils.PlatformServerFlagConstants["Debug"]["ShortFlag"],
		false,
		utils.PlatformServerFlagConstants["Debug"]["Description"])
	fs.BoolP(
		utils.PlatformServerFlagConstants["NoBanner"]["FlagName"],
		utils.PlatformServerFlagConstants["NoBanner"]["ShortFlag"],
		false,
		utils.PlatformServerFlagConstants["NoBanner"]["Description"])
	fs.Bool(
		utils.PlatformServerFlagConstants["Prometheus"]["FlagName"],
		false,
		utils.PlatformServerFlagConstants["Prometheus"]["Description"])
	fs.Int64(
		utils.PlatformServerFlagConstants["RestBridgeTimeout"]["FlagName"],
		1,
		utils.PlatformServerFlagConstants["RestBridgeTimeout"]["Description"])
}

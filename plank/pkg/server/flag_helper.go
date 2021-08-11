// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"flag"
	"fmt"
	"github.com/vmware/transport-go/plank/utils"
	"os"
)

type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	return fmt.Sprint(*f)
}

func (f *stringSliceFlag) Set(value string) error {
	if f == nil {
		*f = make([]string, 0)
	}
	*f = append(*f, value)
	return nil
}

type serverConfigFactory struct {
	HostnamePtr           *string
	PortPtr               *int
	RootDirPtr            *string
	CertPtr               *string
	CertKeyPtr            *string
	StaticPtr             stringSliceFlag
	SpaPathPtr            *string
	NoFabricBrokerPtr     *bool
	FabricEndpointPtr     *string
	TopicPrefixPtr        *string
	QueuePrefixPtr        *string
	RequestPrefixPtr      *string
	RequestQueuePrefixPtr *string
	ConfigFilePtr         *string
	ShutdownTimeoutPtr    *int64
	OutputLogPtr          *string
	AccessLogPtr          *string
	ErrorLogPtr           *string
	DebugPtr              *bool
	NoBannerPtr           *bool
	PrometheusPtr         *bool
	RestBridgeTimeoutPtr *int64

	flagsParsed bool
}

func (f *serverConfigFactory) Hostname() string {
	return *f.HostnamePtr
}

func (f *serverConfigFactory) Port() int {
	return *f.PortPtr
}

func (f *serverConfigFactory) RootDir() string {
	return *f.RootDirPtr
}

func (f *serverConfigFactory) Cert() string {
	return *f.CertPtr
}

func (f *serverConfigFactory) CertKey() string {
	return *f.CertKeyPtr
}

func (f *serverConfigFactory) Static() []string {
	return f.StaticPtr
}

func (f *serverConfigFactory) SpaPath() string {
	return *f.SpaPathPtr
}

func (f *serverConfigFactory) NoFabricBroker() bool {
	return *f.NoFabricBrokerPtr
}

func (f *serverConfigFactory) FabricEndpoint() string {
	return *f.FabricEndpointPtr
}

func (f *serverConfigFactory) TopicPrefix() string {
	return *f.TopicPrefixPtr
}

func (f *serverConfigFactory) QueuePrefix() string {
	return *f.QueuePrefixPtr
}

func (f *serverConfigFactory) RequestPrefix() string {
	return *f.RequestPrefixPtr
}

func (f *serverConfigFactory) RequestQueuePrefix() string {
	return *f.RequestQueuePrefixPtr
}

func (f *serverConfigFactory) ConfigFile() string {
	return *f.ConfigFilePtr
}

func (f *serverConfigFactory) ShutdownTimeout() int64 {
	return *f.ShutdownTimeoutPtr
}

func (f *serverConfigFactory) OutputLog() string {
	return *f.OutputLogPtr
}

func (f *serverConfigFactory) AccessLog() string {
	return *f.AccessLogPtr
}

func (f *serverConfigFactory) ErrorLog() string {
	return *f.ErrorLogPtr
}

func (f *serverConfigFactory) Debug() bool {
	return *f.DebugPtr
}

func (f *serverConfigFactory) NoBanner() bool {
	return *f.NoBannerPtr
}

func (f *serverConfigFactory) Prometheus() bool {
	return *f.PrometheusPtr
}

func (f *serverConfigFactory) RestBridgeTimeout() int64 {
	return *f.RestBridgeTimeoutPtr
}

func (f *serverConfigFactory) parseFlags() {
	flag.Parse()
	f.flagsParsed = flag.Parsed()
}

func (f *serverConfigFactory) configureFlags() {
	wd, _ := os.Getwd()
	f.HostnamePtr = flag.String(utils.PlatformServerFlagConstants["Hostname"]["FlagName"], "localhost", utils.PlatformServerFlagConstants["Hostname"]["Description"])
	f.PortPtr = flag.Int(utils.PlatformServerFlagConstants["Port"]["FlagName"], 30080, utils.PlatformServerFlagConstants["Port"]["Description"])
	f.RootDirPtr = flag.String(utils.PlatformServerFlagConstants["RootDir"]["FlagName"], wd, utils.PlatformServerFlagConstants["RootDir"]["Description"])
	f.CertPtr = flag.String(utils.PlatformServerFlagConstants["Cert"]["FlagName"], "", utils.PlatformServerFlagConstants["Cert"]["Description"])
	f.CertKeyPtr = flag.String(utils.PlatformServerFlagConstants["CertKey"]["FlagName"], "", utils.PlatformServerFlagConstants["CertKey"]["Description"])
	flag.Var(&f.StaticPtr, utils.PlatformServerFlagConstants["Static"]["FlagName"], utils.PlatformServerFlagConstants["Static"]["Description"])
	f.SpaPathPtr = flag.String(utils.PlatformServerFlagConstants["SpaPath"]["FlagName"], "", utils.PlatformServerFlagConstants["SpaPath"]["Description"])
	f.NoFabricBrokerPtr = flag.Bool(utils.PlatformServerFlagConstants["NoFabricBroker"]["FlagName"], false, utils.PlatformServerFlagConstants["NoFabricBroker"]["Description"])
	f.FabricEndpointPtr = flag.String(utils.PlatformServerFlagConstants["FabricEndpoint"]["FlagName"], "/ws", utils.PlatformServerFlagConstants["FabricEndpoint"]["Description"])
	f.TopicPrefixPtr = flag.String(utils.PlatformServerFlagConstants["TopicPrefix"]["FlagName"], "/topic", utils.PlatformServerFlagConstants["TopicPrefix"]["Description"])
	f.QueuePrefixPtr = flag.String(utils.PlatformServerFlagConstants["QueuePrefix"]["FlagName"], "/queue", utils.PlatformServerFlagConstants["QueuePrefix"]["Description"])
	f.RequestPrefixPtr = flag.String(utils.PlatformServerFlagConstants["RequestPrefix"]["FlagName"], "/pub", utils.PlatformServerFlagConstants["RequestPrefix"]["Description"])
	f.RequestQueuePrefixPtr = flag.String(utils.PlatformServerFlagConstants["RequestQueuePrefix"]["FlagName"], "/pub/queue", utils.PlatformServerFlagConstants["RequestQueuePrefix"]["Description"])
	f.ConfigFilePtr = flag.String(utils.PlatformServerFlagConstants["ConfigFile"]["FlagName"], "", utils.PlatformServerFlagConstants["ConfigFile"]["Description"])
	f.ShutdownTimeoutPtr = flag.Int64(utils.PlatformServerFlagConstants["ShutdownTimeout"]["FlagName"], 5, utils.PlatformServerFlagConstants["ShutdownTimeout"]["Description"])
	f.OutputLogPtr = flag.String(utils.PlatformServerFlagConstants["OutputLog"]["FlagName"], "stdout", utils.PlatformServerFlagConstants["OutputLog"]["Description"])
	f.AccessLogPtr = flag.String(utils.PlatformServerFlagConstants["AccessLog"]["FlagName"], "stdout", utils.PlatformServerFlagConstants["AccessLog"]["Description"])
	f.ErrorLogPtr = flag.String(utils.PlatformServerFlagConstants["ErrorLog"]["FlagName"], "stderr", utils.PlatformServerFlagConstants["ErrorLog"]["Description"])
	f.DebugPtr = flag.Bool(utils.PlatformServerFlagConstants["Debug"]["FlagName"], false, utils.PlatformServerFlagConstants["Debug"]["Description"])
	f.NoBannerPtr = flag.Bool(utils.PlatformServerFlagConstants["NoBanner"]["FlagName"], false, utils.PlatformServerFlagConstants["NoBanner"]["Description"])
	f.PrometheusPtr = flag.Bool(utils.PlatformServerFlagConstants["Prometheus"]["FlagName"], false, utils.PlatformServerFlagConstants["Prometheus"]["Description"])
	f.RestBridgeTimeoutPtr = flag.Int64(utils.PlatformServerFlagConstants["RestBridgeTimeout"]["FlagName"], 1, utils.PlatformServerFlagConstants["RestBridgeTimeout"]["Description"])
}

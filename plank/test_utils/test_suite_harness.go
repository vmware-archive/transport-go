// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package test_utils

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/plank/pkg/server"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"os"
	"time"
)

// PlankIntegrationTestSuite makes it easy to set up integration tests for services, that use plank and transport
// In a realistic manner. This is a convenience mechanism to avoid having to rig this harnessing up yourself.
type PlankIntegrationTestSuite struct {

	// suite.Suite is a reference to a testify test suite.
	suite.Suite

	// server.PlatformServer is a reference to plank
	server.PlatformServer

	// os.Signal is the signal channel passed into plan for OS level notifications.
	Syschan chan os.Signal

	// bus.ChannelManager is a reference to the Transport's Channel Manager.
	bus.ChannelManager

	// bus.EventBus is a reference to Transport.
	bus.EventBus
}

// PlankIntegrationTest allows test suites that use PlankIntegrationTestSuite as an embedded struct to set everything
// we need on our test. This is the only contract required to use this harness.
type PlankIntegrationTest interface {
	SetPlatformServer(server.PlatformServer)
	SetSysChan(chan os.Signal)
	SetChannelManager(bus.ChannelManager)
	SetBus(eventBus bus.EventBus)
}

// SetupPlankTestSuiteForTest will copy over everything from the newly set up test suite, to a suite being run.
func SetupPlankTestSuiteForTest(suite *PlankIntegrationTestSuite, test PlankIntegrationTest) {
	test.SetPlatformServer(suite.PlatformServer)
	test.SetSysChan(suite.Syschan)
	test.SetChannelManager(suite.ChannelManager)
	test.SetBus(suite.EventBus)
}

// GetBasicTestServerConfig will generate a simple platform server config, ready to use in a test.
func GetBasicTestServerConfig(rootDir, outLog, accessLog, errLog string, port int, noBanner bool) *server.PlatformServerConfig {
	cfg := &server.PlatformServerConfig{
		RootDir:                    rootDir,
		Host:                       "localhost",
		Port:                       port,
		RestBridgeTimeoutInMinutes: time.Minute,
		LogConfig: &utils.LogConfig{
			OutputLog:     outLog,
			AccessLog:     accessLog,
			ErrorLog:      errLog,
			FormatOptions: &utils.LogFormatOption{},
		},
		NoBanner:                 noBanner,
		ShutdownTimeoutInMinutes: 1,
	}
	return cfg
}

// SetupPlankTestSuite will boot a new instance of plank on your chosen port and will also fire up your service
// Ready to be tested. This always runs on localhost.
func SetupPlankTestSuite(service service.FabricService, serviceChannel string, port int,
	config *server.PlatformServerConfig) (*PlankIntegrationTestSuite, error) {

	s := &PlankIntegrationTestSuite{}

	customFormatter := new(logrus.TextFormatter)
	utils.Log.SetFormatter(customFormatter)
	customFormatter.DisableTimestamp = true

	// check if config has been supplied, if not, generate default one.
	if config == nil {
		config = GetBasicTestServerConfig("/", "stdout", "stdout", "stderr", port, true)
	}

	s.PlatformServer = server.NewPlatformServer(config)
	if err := s.PlatformServer.RegisterService(service, serviceChannel); err != nil {
		return nil, errors.New("cannot create printing press service, test failed")
	}

	s.Syschan = make(chan os.Signal, 1)
	go s.PlatformServer.StartServer(s.Syschan)

	s.EventBus = bus.GetBus()

	// get a pointer to the channel manager
	s.ChannelManager = s.EventBus.GetChannelManager()

	// wait, service may be slow loading, rest mapping happens last.
	wait := make(chan bool)
	go func() {
		time.Sleep(10 * time.Millisecond)
		wait <- true // Sending something to the channel to let the main thread continue

	}()

	<-wait
	return s, nil
}

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
// we need on our test.
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

func SetupPlankTestSuite(service service.FabricService, serviceChannel string, port int) (*PlankIntegrationTestSuite, error) {

	s := &PlankIntegrationTestSuite{}

	customFormatter := new(logrus.TextFormatter)
	utils.Log.SetFormatter(customFormatter)
	customFormatter.DisableTimestamp = true

	// instantiate a server config
	serverConfig := &server.PlatformServerConfig{
		Host:                       "localhost",
		NoBanner:                   true,
		Port:                       port,
		RootDir:                    "/",
		StaticDir:                  []string{"/static"},
		RestBridgeTimeoutInMinutes: 30 * time.Second,
		ShutdownTimeoutInMinutes:   1,
		LogConfig: &utils.LogConfig{
			OutputLog:     "stdout",
			AccessLog:     "stdout",
			ErrorLog:      "stderr",
			FormatOptions: &utils.LogFormatOption{DisableTimestamp: true},
		},
	}

	s.PlatformServer = server.NewPlatformServer(serverConfig)
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

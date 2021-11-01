// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	svc "github.com/vmware/transport-go/service"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testSuitePortMap = make(map[string]int)

// PlankIntegrationTestSuite makes it easy to set up integration tests for services, that use plank and transport
// In a realistic manner. This is a convenience mechanism to avoid having to rig this harnessing up yourself.
type PlankIntegrationTestSuite struct {

	// suite.Suite is a reference to a testify test suite.
	suite.Suite

	// server.PlatformServer is a reference to plank
	PlatformServer

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
	SetPlatformServer(PlatformServer)
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
func GetBasicTestServerConfig(rootDir, outLog, accessLog, errLog string, port int, noBanner bool) *PlatformServerConfig {
	cfg := &PlatformServerConfig{
		RootDir:           rootDir,
		Host:              "localhost",
		Port:              port,
		RestBridgeTimeout: time.Minute,
		LogConfig: &utils.LogConfig{
			OutputLog:     outLog,
			AccessLog:     accessLog,
			ErrorLog:      errLog,
			FormatOptions: &utils.LogFormatOption{},
		},
		NoBanner:        noBanner,
		ShutdownTimeout: time.Minute,
	}
	return cfg
}

// SetupPlankTestSuite will boot a new instance of plank on your chosen port and will also fire up your service
// Ready to be tested. This always runs on localhost.
func SetupPlankTestSuite(service svc.FabricService, serviceChannel string, port int,
	config *PlatformServerConfig) (*PlankIntegrationTestSuite, error) {

	s := &PlankIntegrationTestSuite{}

	customFormatter := new(logrus.TextFormatter)
	utils.Log.SetFormatter(customFormatter)
	customFormatter.DisableTimestamp = true

	// check if config has been supplied, if not, generate default one.
	if config == nil {
		config = GetBasicTestServerConfig("/", "stdout", "stdout", "stderr", port, true)
	}

	s.PlatformServer = NewPlatformServer(config)
	if err := s.PlatformServer.RegisterService(service, serviceChannel); err != nil {
		return nil, errors.New("cannot create printing press service, test failed")
	}

	s.Syschan = make(chan os.Signal, 1)
	go s.PlatformServer.StartServer(s.Syschan)

	s.EventBus = bus.ResetBus()
	svc.ResetServiceRegistry()

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

// CreateTestServer consumes *server.PlatformServerConfig and returns a new instance of PlatformServer along with
// its base URL and log file name
func CreateTestServer(config *PlatformServerConfig) (baseUrl, logFile string, s PlatformServer) {
	testServer := NewPlatformServer(config)

	protocol := "http"
	if config.TLSCertConfig != nil {
		protocol += "s"
	}

	baseUrl = fmt.Sprintf("%s://%s:%d", protocol, config.Host, config.Port)
	return baseUrl, config.LogConfig.OutputLog, testServer
}

// RunWhenServerReady runs test function fn after Plank has booted up
func RunWhenServerReady(t *testing.T, eventBus bus.EventBus, fn func(*testing.T)) {
	handler, _ := eventBus.ListenOnce(PLANK_SERVER_ONLINE_CHANNEL)
	handler.Handle(func(message *model.Message) {
		fn(t)
	}, func(err error) {
		assert.FailNow(t, err.Error())
	})
}

// GetTestPort returns an available port for use by Plank tests
func GetTestPort() int {
	minPort := 9980
	fr := utils.GetCallerStackFrame()
	port, exists := testSuitePortMap[fr.File]
	if exists {
		return port
	}

	// try 5 more times with every failure advancing port by one
	tryPort := minPort
	for i := 0; i <= 4; i++ {
		tryPort = minPort + len(testSuitePortMap) + i
		_, err := net.Dial("tcp", fmt.Sprintf(":%d", tryPort))
		if err == nil { // port in use, try next one
			continue
		}

		testSuitePortMap[fr.File] = tryPort
		break
	}

	if testSuitePortMap[fr.File] == 0 { // port could not be assigned
		panic(fmt.Errorf("could not assign a port for tests. last tried port is %d", tryPort))
	}

	return testSuitePortMap[fr.File]
}

// CreateConfigJsonForTest creates and returns the path to a file containing the plank configuration in JSON format
func CreateConfigJsonForTest() (string, error) {
	configJsonContent := `{
  "debug": true,
  "no_banner": true,
  "root_dir": "./",
  "host": "localhost",
  "port": 31234,
  "log_config": {
    "root": ".",
    "access_log": "access.log",
    "error_log": "errors.log",
    "output_log": "stdout",
    "format_options": {
      "force_colors": false,
      "disable_colors": true,
      "force_quote": false,
      "disable_quote": false,
      "environment_override_colors": false,
      "disable_timestamp": false,
      "full_timestamp": true,
      "timestamp_format": "",
      "disable_sorting": false,
      "disable_level_truncation": false,
      "pad_level_text": false,
      "quote_empty_fields": false,
      "is_terminal": true
    }
  },
  "spa_config": {
    "root_folder": "public/",
    "base_uri": "/",
    "static_assets": [
      "public/assets:/assets"
    ],
    "cache_control_rules": {
      "*.{js,css}": "public, max-age=86400",
      "*.{ico,jpg,jpeg,svg,png,gif,tiff}": "public, max-age=604800"
    }
  },
  "shutdown_timeout_in_minutes": -1,
  "rest_bridge_timeout_in_minutes": 0,
  "fabric_config": {
    "fabric_endpoint": "/ws",
    "endpoint_config": {
      "TopicPrefix": "/topic",
      "UserQueuePrefix": "/queue",
      "AppRequestPrefix": "/pub",
      "AppRequestQueuePrefix": "/pub/queue",
      "Heartbeat": 60000
    }
  },
  "enable_prometheus": true,
  "tls_config": {
    "cert_file": "cert/fullchain.pem",
    "key_file": "cert/server.key"
  }
}`
	testDir := filepath.Join(os.TempDir(), "plank-tests")
	testFile := filepath.Join(testDir, "test-config.json")
	err := os.MkdirAll(testDir, 0744)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(testFile, []byte(configJsonContent), 0744)
	if err != nil {
		return "", err
	}
	return testFile, nil
}

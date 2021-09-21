package server

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testSuitePortMap = make(map[string]int)

func getBasicTestServerConfig(rootDir, outLog, accessLog, errLog string, port int, noBanner bool) *PlatformServerConfig {
	cfg := &PlatformServerConfig{
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

func createTestServer(config *PlatformServerConfig) (baseUrl, logFile string, s PlatformServer) {
	testServer := NewPlatformServer(config)

	protocol := "http"
	if config.TLSCertConfig != nil {
		protocol += "s"
	}

	baseUrl = fmt.Sprintf("%s://%s:%d", protocol, config.Host, config.Port)
	return baseUrl, config.LogConfig.OutputLog, testServer
}

func runWhenServerReady(t *testing.T, eventBus bus.EventBus, fn func(t *testing.T)) {
	handler, _ := eventBus.ListenOnce(PLANK_SERVER_ONLINE_CHANNEL)
	handler.Handle(func(message *model.Message) {
		fn(t)
	}, func(err error) {
		assert.FailNow(t, err.Error())
	})
}

func getTestPort() int {
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

func createConfigJsonForTest() (string, error) {
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

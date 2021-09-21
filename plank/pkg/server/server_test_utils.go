package server

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"net"
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
		NoBanner: noBanner,
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

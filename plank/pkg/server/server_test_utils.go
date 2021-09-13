package server

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"sync"
	"testing"
	"time"
)

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

func runTestMainWhenServerReady(m *testing.M, eventBus bus.EventBus) int {
	wg := sync.WaitGroup{}
	wg.Add(1)
	var exitCode int
	handler, _ := eventBus.ListenOnce(PLANK_SERVER_ONLINE_CHANNEL)
	handler.Handle(func(message *model.Message) {
		exitCode = m.Run()
		wg.Done()
	}, func(err error) {})
	wg.Wait()
	return exitCode
}

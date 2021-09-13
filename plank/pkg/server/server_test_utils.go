package server

import (
	"fmt"
	"github.com/vmware/transport-go/plank/utils"
	"time"
)

func getBasicTestServerConfig(rootDir, outLog, accessLog, errLog string, port int) *PlatformServerConfig {
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
		NoBanner: true,
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

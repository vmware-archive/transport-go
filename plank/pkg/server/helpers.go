package server

import (
	"encoding/json"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/plank/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// generatePlatformServerConfig is a generic internal method that returns the pointer of a new
// instance of PlatformServerConfig. for an argument it can be passed either *serverConfigFactory
// or *cli.Context which the method will analyze and determine the best way to extract user provided values from it.
func generatePlatformServerConfig(f *serverConfigFactory) (*PlatformServerConfig, error) {
	configFile := f.ConfigFile()
	host := f.Hostname()
	port := f.Port()
	rootDir := f.RootDir()
	static := f.Static()
	shutdownTimeoutInMinutes := f.ShutdownTimeout()
	accessLog := f.AccessLog()
	outputLog := f.OutputLog()
	errorLog := f.ErrorLog()
	debug := f.Debug()
	noBanner := f.NoBanner()
	cert := f.Cert()
	certKey := f.CertKey()
	spaPath := f.SpaPath()
	noFabricBroker := f.NoFabricBroker()
	fabricEndpoint := f.FabricEndpoint()
	topicPrefix := f.TopicPrefix()
	queuePrefix := f.QueuePrefix()
	requestPrefix := f.RequestPrefix()
	requestQueuePrefix := f.RequestQueuePrefix()
	prometheus := f.Prometheus()
	restBridgeTimeout := f.RestBridgeTimeout()

	// if config file flag is provided, read directly from the file
	if len(configFile) > 0 {
		var serverConfig PlatformServerConfig
		b, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &serverConfig); err != nil {
			return nil, err
		}

		// handle invalid duration by setting it to the default value of 5 minutes
		if serverConfig.ShutdownTimeout <= 0 {
			serverConfig.ShutdownTimeout = 5
		}

		// handle invalid duration by setting it to the default value of 1 minute
		if serverConfig.RestBridgeTimeout <= 0 {
			serverConfig.RestBridgeTimeout = 1
		}

		// the raw value from the config.json needs to be multiplied by time.Minute otherwise it's interpreted as nanosecond
		serverConfig.ShutdownTimeout = serverConfig.ShutdownTimeout * time.Minute

		// the raw value from the config.json needs to be multiplied by time.Minute otherwise it's interpreted as nanosecond
		serverConfig.RestBridgeTimeout = serverConfig.RestBridgeTimeout * time.Minute

		// convert map of cache control rules of SpaConfig into an array
		if serverConfig.SpaConfig != nil {
			serverConfig.SpaConfig.CollateCacheControlRules()
		}

		return &serverConfig, nil
	}

	// handle invalid duration by setting it to the default value of 1 minute
	if restBridgeTimeout <= 0 {
		restBridgeTimeout = 1
	}

	// handle invalid duration by setting it to the default value of 5 minutes
	if shutdownTimeoutInMinutes <= 0 {
		shutdownTimeoutInMinutes = 5
	}

	// instantiate a server config
	serverConfig := &PlatformServerConfig{
		Host:            host,
		Port:            port,
		RootDir:         rootDir,
		StaticDir:       static,
		ShutdownTimeout: time.Duration(shutdownTimeoutInMinutes) * time.Minute,
		LogConfig: &utils.LogConfig{
			AccessLog:     accessLog,
			ErrorLog:      errorLog,
			OutputLog:     outputLog,
			Root:          rootDir,
			FormatOptions: &utils.LogFormatOption{},
		},
		Debug:             debug,
		NoBanner:          noBanner,
		EnablePrometheus:  prometheus,
		RestBridgeTimeout: time.Duration(restBridgeTimeout) * time.Minute,
	}

	if len(certKey) > 0 && len(certKey) > 0 {
		var err error
		certKey, err = filepath.Abs(certKey)
		if err != nil {
			return nil, err
		}
		cert, err = filepath.Abs(cert)
		if err != nil {
			return nil, err
		}

		serverConfig.TLSCertConfig = &TLSCertConfig{CertFile: cert, KeyFile: certKey}
	}

	if len(strings.TrimSpace(spaPath)) > 0 {
		var err error
		serverConfig.SpaConfig, err = NewSpaConfig(spaPath)
		if err != nil {
			return nil, err
		}
	}

	// unless --no-Fabric-broker flag is provided, set up a broker config
	if !noFabricBroker {
		serverConfig.FabricConfig = &FabricBrokerConfig{
			FabricEndpoint: fabricEndpoint,
			EndpointConfig: &bus.EndpointConfig{
				TopicPrefix:           topicPrefix,
				UserQueuePrefix:       queuePrefix,
				AppRequestPrefix:      requestPrefix,
				AppRequestQueuePrefix: requestQueuePrefix,
				Heartbeat:             60000},
		}
	}

	return serverConfig, nil
}

// ensureResponseInByteSlice takes body as an interface not knowing whether it is already converted to []byte or not.
// if it is not of []byte type it marshals the payload using json.Marshal. otherwise, the input
// byte slice is returned as-is.
func ensureResponseInByteSlice(body interface{}) (bytes []byte, err error) {
	switch body.(type) {
	case []byte:
		bytes, err = body.([]byte), nil
	default:
		bytes, err = json.Marshal(body)
	}
	return
}

// sanitizeConfigRootPath takes *PlatformServerConfig, ensures the path specified by RootDir field exists.
// if RootDir is empty then the current working directory will be populated. if for some reason the path
// cannot be accessed it'll cause a panic.
func sanitizeConfigRootPath(config *PlatformServerConfig) {
	if len(config.RootDir) == 0 {
		wd, _ := os.Getwd()
		config.RootDir = wd
	}

	absRootPath, err := filepath.Abs(config.RootDir)
	if err != nil {
		panic(err)
	}

	_, err = os.Stat(absRootPath)
	if err != nil {
		panic(err)
	}

	// once it has been confirmed that the path exists, set config.RootDir to the absolute path
	config.RootDir = absRootPath
}

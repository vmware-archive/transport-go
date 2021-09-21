package server

import (
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGeneratePlatformServerConfig_Default(t *testing.T) {
	// arrange
	f := &serverConfigFactory{}
	pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)

	// act
	testArgs := []string{""}
	f.configureFlags(pflag.CommandLine)
	f.parseFlags(testArgs)
	config, err := generatePlatformServerConfig(f)

	// assert
	wd, _ := os.Getwd()
	assert.Nil(t, err)
	assert.EqualValues(t, "localhost", config.Host)
	assert.EqualValues(t, 30080, config.Port)
	assert.EqualValues(t, wd, config.RootDir)
	assert.Empty(t, config.StaticDir)
	assert.EqualValues(t, 5*time.Minute, config.ShutdownTimeoutInMinutes)
	assert.EqualValues(t, "stdout", config.LogConfig.OutputLog)
	assert.EqualValues(t, "stdout", config.LogConfig.AccessLog)
	assert.EqualValues(t, "stderr", config.LogConfig.ErrorLog)
	assert.EqualValues(t, wd, config.LogConfig.Root)
	assert.False(t, config.Debug)
	assert.False(t, config.NoBanner)
	assert.False(t, config.EnablePrometheus)
	assert.EqualValues(t, time.Minute, config.RestBridgeTimeoutInMinutes)
	assert.EqualValues(t, "/ws", config.FabricConfig.FabricEndpoint)
	assert.EqualValues(t, "/topic", config.FabricConfig.EndpointConfig.TopicPrefix)
	assert.EqualValues(t, "/queue", config.FabricConfig.EndpointConfig.UserQueuePrefix)
	assert.EqualValues(t, "/pub", config.FabricConfig.EndpointConfig.AppRequestPrefix)
	assert.EqualValues(t, "/pub/queue", config.FabricConfig.EndpointConfig.AppRequestQueuePrefix)
	assert.EqualValues(t, 60000, config.FabricConfig.EndpointConfig.Heartbeat)
}

func TestGeneratePlatformServerConfig_CertConfig(t *testing.T) {
	// arrange
	f := &serverConfigFactory{}
	pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
	dummyCert := filepath.Join(os.TempDir(), "plank-tests", "cert.pem")
	dummyKey := filepath.Join(os.TempDir(), "plank-tests", "key.pem")

	// act
	testArgs := []string{"", "--cert", dummyCert, "--cert-key", dummyKey}
	f.configureFlags(pflag.CommandLine)
	f.parseFlags(testArgs)
	config, err := generatePlatformServerConfig(f)

	// assert
	assert.Nil(t, err)
	assert.EqualValues(t, dummyCert, config.TLSCertConfig.CertFile)
	assert.EqualValues(t, dummyKey, config.TLSCertConfig.KeyFile)
}

func TestGeneratePlatformServerConfig_SpaConfig(t *testing.T) {
	// arrange
	f := &serverConfigFactory{}
	pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
	spaRoot := filepath.Join(os.TempDir(), "plank-tests", "spaRoot")

	// act
	testArgs := []string{"", "--spa-path", spaRoot}
	f.configureFlags(pflag.CommandLine)
	f.parseFlags(testArgs)
	config, err := generatePlatformServerConfig(f)

	// assert
	assert.Nil(t, err)
	assert.EqualValues(t, spaRoot, config.SpaConfig.RootFolder)
	assert.EqualValues(t, "spaRoot", config.SpaConfig.BaseUri)
}

func TestGeneratePlatformServerConfig_ConfigFile(t *testing.T) {
	// arrange
	f := &serverConfigFactory{}
	pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
	configFile, err := CreateConfigJsonForTest()
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	defer os.RemoveAll(filepath.Dir(configFile))

	// act
	testArgs := []string{"", "--config-file", configFile}
	f.configureFlags(pflag.CommandLine)
	f.parseFlags(testArgs)
	config, err := generatePlatformServerConfig(f)

	// assert
	assert.Nil(t, err)
	assert.EqualValues(t, "localhost", config.Host)
	assert.EqualValues(t, 31234, config.Port)
	assert.EqualValues(t, "./", config.RootDir)
	assert.Empty(t, config.StaticDir)
	assert.EqualValues(t, 1*time.Minute, config.ShutdownTimeoutInMinutes)
	assert.EqualValues(t, "stdout", config.LogConfig.OutputLog)
	assert.EqualValues(t, "access.log", config.LogConfig.AccessLog)
	assert.EqualValues(t, "errors.log", config.LogConfig.ErrorLog)
	assert.EqualValues(t, ".", config.LogConfig.Root)
	assert.True(t, config.Debug)
	assert.True(t, config.NoBanner)
	assert.True(t, config.EnablePrometheus)
	assert.EqualValues(t, time.Minute, config.RestBridgeTimeoutInMinutes)
	assert.EqualValues(t, "cert/server.key", config.TLSCertConfig.KeyFile)
	assert.EqualValues(t, "cert/fullchain.pem", config.TLSCertConfig.CertFile)
	assert.EqualValues(t, "/ws", config.FabricConfig.FabricEndpoint)
	assert.EqualValues(t, "/topic", config.FabricConfig.EndpointConfig.TopicPrefix)
	assert.EqualValues(t, "/queue", config.FabricConfig.EndpointConfig.UserQueuePrefix)
	assert.EqualValues(t, "/pub", config.FabricConfig.EndpointConfig.AppRequestPrefix)
	assert.EqualValues(t, "/pub/queue", config.FabricConfig.EndpointConfig.AppRequestQueuePrefix)
	assert.EqualValues(t, 60000, config.FabricConfig.EndpointConfig.Heartbeat)
	assert.EqualValues(t, "public/", config.SpaConfig.RootFolder)
	assert.EqualValues(t, "/", config.SpaConfig.BaseUri)
	assert.EqualValues(t, "public/assets:/assets", config.SpaConfig.StaticAssets[0])
}

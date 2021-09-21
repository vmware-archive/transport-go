package server

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestFlagHelper_ParseDefaultFlags(t *testing.T) {
	// arrange
	f := &serverConfigFactory{}
	pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)

	// act
	testArgs := []string{""}
	f.configureFlags(pflag.CommandLine)
	f.parseFlags(testArgs)

	// assert
	wd, _ := os.Getwd()
	assert.True(t, f.flagsParsed)
	assert.EqualValues(t, "localhost", f.Hostname())
	assert.EqualValues(t, 30080, f.Port())
	assert.EqualValues(t, wd, f.RootDir())
	assert.Empty(t, f.Cert())
	assert.Empty(t, f.CertKey())
	assert.Empty(t, f.Static())
	assert.Empty(t, f.SpaPath())
	assert.False(t, f.NoFabricBroker())
	assert.EqualValues(t, "/ws", f.FabricEndpoint())
	assert.EqualValues(t, "/topic", f.TopicPrefix())
	assert.EqualValues(t, "/queue", f.QueuePrefix())
	assert.EqualValues(t, "/pub", f.RequestPrefix())
	assert.EqualValues(t, "/pub/queue", f.RequestQueuePrefix())
	assert.Empty(t, f.ConfigFile())
	assert.EqualValues(t, 5, f.ShutdownTimeout())
	assert.EqualValues(t, "stdout", f.OutputLog())
	assert.EqualValues(t, "stdout", f.AccessLog())
	assert.EqualValues(t, "stderr", f.ErrorLog())
	assert.False(t, f.Debug())
	assert.False(t, f.NoBanner())
	assert.False(t, f.Prometheus())
	assert.EqualValues(t, 1, f.RestBridgeTimeout())
}

func TestFlagHelper_ParseFlags(t *testing.T) {
	// arrange
	f := &serverConfigFactory{}
	pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)

	// act
	testArgs := []string{"", "--hostname", "cloud.local", "--port", "443", "--static", "test", "--static", "test2"}
	f.configureFlags(pflag.CommandLine)
	f.parseFlags(testArgs)

	// assert
	assert.True(t, f.flagsParsed)
	assert.EqualValues(t, "cloud.local", f.Hostname())
	assert.EqualValues(t, 443, f.Port())
	assert.Len(t, f.Static(), 2)
	assert.EqualValues(t, "[test test2]", fmt.Sprint(f.Static()))
}

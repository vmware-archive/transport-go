package server

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestPrintBanner_BasicBootMessage(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", 9981, false)
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)
	testServer.printBanner()

	// visually inspect the following fields are printed out: host, port, health endpoint
}

func TestPrintBanner_BasicBootMessage_NonStdout(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	testLogFile := filepath.Join(testRoot, "testlog.log")
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, testLogFile, testLogFile, testLogFile, 9981, false)
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// act
	testServer.printBanner()

	// assert
	logContents, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.FileExists(t, testLogFile)
	assert.Contains(t, string(logContents), "P L A N K")
	assert.Contains(t, string(logContents), "Host\t\t\tlocalhost")
	assert.Contains(t, string(logContents), "Port\t\t\t9981")
	assert.Contains(t, string(logContents), "Health endpoint\t\t/health")
}

func TestPrintBanner_StaticConfig(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	testLogFile := filepath.Join(testRoot, "testlog.log")
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, testLogFile, testLogFile, testLogFile, 9981, false)
	cfg.StaticDir = []string{"somewhere/over-the-rainbow"}
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// act
	testServer.printBanner()

	// assert
	logContents, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.FileExists(t, testLogFile)
	assert.Contains(t, string(logContents), "Host\t\t\tlocalhost")
	assert.Contains(t, string(logContents), "Port\t\t\t9981")
	assert.Contains(t, string(logContents), "Static endpoints\t/over-the-rainbow")
}

func TestPrintBanner_FabricConfig(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	testLogFile := filepath.Join(testRoot, "testlog.log")
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, testLogFile, testLogFile, testLogFile, 9981, false)
	cfg.FabricConfig = &FabricBrokerConfig{
		FabricEndpoint: "/ws",
		EndpointConfig: &bus.EndpointConfig{
			TopicPrefix:           "/topic",
			UserQueuePrefix:       "/queue",
			AppRequestPrefix:      "/pub/topic",
			AppRequestQueuePrefix: "/pub/queue",
			Heartbeat:             30000,
		},
	}
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// act
	testServer.printBanner()

	// assert
	logContents, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.FileExists(t, testLogFile)
	assert.Contains(t, string(logContents), "Host\t\t\tlocalhost")
	assert.Contains(t, string(logContents), "Port\t\t\t9981")
	assert.Contains(t, string(logContents), "Fabric endpoint\t\t/ws")
}

func TestPrintBanner_FabricConfig_TCP(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	testLogFile := filepath.Join(testRoot, "testlog.log")
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, testLogFile, testLogFile, testLogFile, 9981, false)
	cfg.FabricConfig = &FabricBrokerConfig{
		FabricEndpoint: "/ws",
		UseTCP:         true,
		TCPPort:        61613,
		EndpointConfig: &bus.EndpointConfig{
			TopicPrefix:           "/topic",
			UserQueuePrefix:       "/queue",
			AppRequestPrefix:      "/pub/topic",
			AppRequestQueuePrefix: "/pub/queue",
			Heartbeat:             30000,
		},
	}
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// act
	testServer.printBanner()

	// assert
	logContents, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.FileExists(t, testLogFile)
	assert.Contains(t, string(logContents), "Host\t\t\tlocalhost")
	assert.Contains(t, string(logContents), "Port\t\t\t9981")
	assert.Contains(t, string(logContents), "Fabric endpoint\t\t:61613 (TCP)")
}

func TestPrintBanner_SpaConfig(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	testLogFile := filepath.Join(testRoot, "testlog.log")
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, testLogFile, testLogFile, testLogFile, 9981, false)
	cfg.SpaConfig = &SpaConfig{
		RootFolder:        testRoot,
		BaseUri:           "/",
		StaticAssets:      []string{"a", "b", "c:/public/c"},
		CacheControlRules: nil,
	}
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// act
	testServer.printBanner()

	// assert
	logContents, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.FileExists(t, testLogFile)
	assert.Contains(t, string(logContents), "Host\t\t\tlocalhost")
	assert.Contains(t, string(logContents), "Port\t\t\t9981")
	assert.Contains(t, string(logContents), "SPA endpoint\t\t/")
	assert.Contains(t, string(logContents), "SPA static assets\t/a, /b, /public/c")
}

func TestPrintBanner_Prometheus(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	testLogFile := filepath.Join(testRoot, "testlog.log")
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, testLogFile, testLogFile, testLogFile, 9981, false)
	cfg.EnablePrometheus = true
	_, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// act
	testServer.printBanner()

	// assert
	logContents, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.FileExists(t, testLogFile)
	assert.Contains(t, string(logContents), "Host\t\t\tlocalhost")
	assert.Contains(t, string(logContents), "Port\t\t\t9981")
	assert.Contains(t, string(logContents), "Prometheus endpoint\t/prometheus")
}

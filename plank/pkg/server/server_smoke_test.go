package server

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

var eventbus = bus.GetBus()

func TestSmokeTests(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	//testOutFile := filepath.Join(testRoot, "plank-server-tests.log")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	cfg := getBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", 9981, true)
	cfg.NoBanner = true
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
	baseUrl, _, testServer := createTestServer(cfg)

	assert.EqualValues(t, "http://localhost:9981", baseUrl)

	syschan := make(chan os.Signal, 1)
	go testServer.StartServer(syschan)
	runWhenServerReady(t, eventbus, func(t *testing.T) {
		// root url - 404
		t.Run("404 on root", func(t2 *testing.T) {
			cl := http.DefaultClient
			rsp, err := cl.Get(baseUrl)
			assert.Nil(t2, err)
			assert.EqualValues(t2, 404, rsp.StatusCode)
		})

		// connection to fabric endpoint
		t.Run("fabric endpoint should exist", func(t2 *testing.T) {
			cl := http.DefaultClient
			rsp, err := cl.Get(fmt.Sprintf("%s/ws", baseUrl))
			assert.Nil(t2, err)
			assert.EqualValues(t2, 400, rsp.StatusCode)
		})

		// (in separate test suites)
		// 1.
		syschan <- syscall.SIGINT
	})
}

func TestSmokeTests_NoFabric(t *testing.T) {
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	cfg := getBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", 9981, true)
	cfg.FabricConfig = nil
	baseUrl, _, testServer := createTestServer(cfg)

	assert.EqualValues(t, "http://localhost:9981", baseUrl)

	syschan := make(chan os.Signal, 1)
	go testServer.StartServer(syschan)
	runWhenServerReady(t, eventbus, func(t *testing.T) {
		// fabric - 404
		t.Run("404 on fabric endpoint", func(t2 *testing.T) {
			cl := http.DefaultClient
			rsp, err := cl.Get(fmt.Sprintf("%s/ws", baseUrl))
			assert.Nil(t2, err)
			assert.EqualValues(t2, 404, rsp.StatusCode)
		})

		syschan <- syscall.SIGINT
	})
}

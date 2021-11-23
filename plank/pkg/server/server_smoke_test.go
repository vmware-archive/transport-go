package server

import (
	"crypto/tls"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/service"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestSmokeTests_TLS tests if Plank starts with TLS enabled
func TestSmokeTests_TLS(t *testing.T) {
	// pre-arrange
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	// arrange
	port := GetTestPort()
	cfg := GetBasicTestServerConfig(testRoot, "stdout", "null", "stderr", port, true)
	cfg.FabricConfig = GetTestFabricBrokerConfig()
	cfg.TLSCertConfig = GetTestTLSCertConfig(testRoot)

	// act
	var wg sync.WaitGroup
	sigChan := make(chan os.Signal)
	baseUrl, _, testServer := CreateTestServer(cfg)
	testServerInternal := testServer.(*platformServer)
	testServerInternal.setEventBusRef(newBus)

	// assert to make sure the server was created with the correct test arguments
	assert.EqualValues(t, fmt.Sprintf("https://localhost:%d", port), baseUrl)

	wg.Add(1)
	go testServer.StartServer(sigChan)

	originalTransport := http.DefaultTransport
	originalTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	RunWhenServerReady(t, newBus, func(t *testing.T) {
		resp, err := http.Get(baseUrl)
		if err != nil {
			defer func() {
				testServer.StopServer()
				wg.Done()
			}()
			t.Fatal(err)
		}
		assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
		testServer.StopServer()
		wg.Done()
	})
	wg.Wait()
}

// TestSmokeTests_TLS_InvalidCert tests if Plank fails to start because of an invalid cert
func TestSmokeTests_TLS_InvalidCert(t *testing.T) {
	// TODO: make StartServer return an error object so it's easier to test
}

func TestSmokeTests(t *testing.T) {
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	//testOutFile := filepath.Join(testRoot, "plank-server-tests.log")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	port := GetTestPort()
	cfg := GetBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", port, true)
	cfg.NoBanner = true
	cfg.FabricConfig = GetTestFabricBrokerConfig()

	baseUrl, _, testServer := CreateTestServer(cfg)
	testServer.(*platformServer).eventbus = newBus

	assert.EqualValues(t, fmt.Sprintf("http://localhost:%d", port), baseUrl)

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go testServer.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(t *testing.T) {
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

		testServer.StopServer()
		wg.Done()
	})
	wg.Wait()
}

func TestSmokeTests_NoFabric(t *testing.T) {
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	port := GetTestPort()
	cfg := GetBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", port, true)
	cfg.FabricConfig = nil
	baseUrl, _, testServer := CreateTestServer(cfg)
	testServer.(*platformServer).eventbus = newBus

	assert.EqualValues(t, fmt.Sprintf("http://localhost:%d", port), baseUrl)

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go testServer.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(t *testing.T) {
		// fabric - 404
		t.Run("404 on fabric endpoint", func(t2 *testing.T) {
			cl := http.DefaultClient
			rsp, err := cl.Get(fmt.Sprintf("%s/ws", baseUrl))
			assert.Nil(t2, err)
			assert.EqualValues(t2, 404, rsp.StatusCode)
		})

		testServer.StopServer()
		wg.Done()
	})
	wg.Wait()
}

func TestSmokeTests_HealthEndpoint(t *testing.T) {
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	port := GetTestPort()
	cfg := GetBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", port, true)
	cfg.FabricConfig = nil
	baseUrl, _, testServer := CreateTestServer(cfg)
	testServer.(*platformServer).eventbus = newBus

	assert.EqualValues(t, fmt.Sprintf("http://localhost:%d", port), baseUrl)

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go testServer.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(*testing.T) {
		t.Run("/health returns OK", func(t2 *testing.T) {
			cl := http.DefaultClient
			rsp, err := cl.Get(fmt.Sprintf("%s/health", baseUrl))
			assert.Nil(t2, err)
			defer rsp.Body.Close()
			bodyBytes, _ := ioutil.ReadAll(rsp.Body)
			assert.Contains(t, string(bodyBytes), "OK")
		})

		testServer.StopServer()
		wg.Done()
	})
	wg.Wait()
}

package server

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestInitialize_DebugLogging(t *testing.T) {
	// arrange
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)

	cfg := GetBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", GetTestPort(), true)
	cfg.Debug = true

	// act
	_, _, _ = CreateTestServer(cfg)

	// assert
	assert.EqualValues(t, logrus.DebugLevel, utils.Log.GetLevel())
}

func TestInitialize_RestBridgeOverride(t *testing.T) {
	// arrange
	bus.ResetBus()
	service.ResetServiceRegistry()
	testRoot := filepath.Join(os.TempDir(), "plank-tests")
	_ = os.MkdirAll(testRoot, 0755)
	defer os.RemoveAll(testRoot)
	defer service.GetServiceRegistry().UnregisterService(services.PingPongServiceChan)

	cfg := GetBasicTestServerConfig(testRoot, "stdout", "stdout", "stderr", GetTestPort(), true)
	baseUrl, _, testServerInterface := CreateTestServer(cfg)
	testServer := testServerInterface.(*platformServer)

	// register ping pong service with default bridge points of /rest/ping-pong, /rest/ping-pong2 and /rest/ping-pong/{from}/{to}/{message}
	testServerInterface.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)

	// act
	// replace existing rest bridges with a new config
	oldRouter := testServer.router
	time.Sleep(50 * time.Millisecond)
	_ = bus.GetBus().SendResponseMessage(service.LifecycleManagerChannelName, &service.SetupRESTBridgeRequest{
		ServiceChannel: services.PingPongServiceChan,
		Override:       true,
		Config: []*service.RESTBridgeConfig{
			{
				ServiceChannel: services.PingPongServiceChan,
				Uri:            "/ping-new",
				Method:         "GET",
				FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
					return model.Request{Id: &uuid.UUID{}, Request: "ping-get", Payload: r.URL.Query().Get("message")}
				},
			},
		},
	}, bus.GetBus().GetId())

	// start server
	syschan := make(chan os.Signal)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go testServerInterface.StartServer(syschan)

	// assert
	RunWhenServerReady(t, bus.GetBus(), func(t2 *testing.T) {
		time.Sleep(time.Millisecond)
		// router instance should have been swapped
		assert.NotEqual(t, testServer.router, oldRouter)

		// old endpoints should 404
		rsp, err := http.Get(fmt.Sprintf("%s/rest/ping-pong", baseUrl))
		assert.Nil(t, err)
		assert.EqualValues(t, 404, rsp.StatusCode)

		rsp, err = http.Get(fmt.Sprintf("%s/rest/ping-pong2", baseUrl))
		assert.Nil(t, err)
		assert.EqualValues(t, 404, rsp.StatusCode)

		rsp, err = http.Get(fmt.Sprintf("%s/rest/ping-pong/a/b/c", baseUrl))
		assert.Nil(t, err)
		assert.EqualValues(t, 404, rsp.StatusCode)

		// new endpoints should respond successfully
		rsp, err = http.Get(fmt.Sprintf("%s/ping-new", baseUrl))
		assert.Nil(t, err)
		assert.EqualValues(t, 200, rsp.StatusCode)

		syschan <- syscall.SIGINT
		wg.Done()
	})

	wg.Wait()
}

// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/service"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewPlatformServer(t *testing.T) {
	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	assert.NotNil(t, ps)
}

func TestNewPlatformServer_EmptyRootDir(t *testing.T) {
	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	newConfig := GetBasicTestServerConfig("", "stdout", "stdout", "stderr", port, true)
	NewPlatformServer(newConfig)
	wd, _ := os.Getwd()
	assert.Equal(t, wd, newConfig.RootDir)
}

func TestNewPlatformServer_FileLog(t *testing.T) {
	defer func() {
		_ = os.Remove(filepath.Join(os.TempDir(), "testlog.log"))
	}()

	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	newConfig := GetBasicTestServerConfig(os.TempDir(), filepath.Join(os.TempDir(), "testlog.log"), "stdout", "stderr", port, true)
	NewPlatformServer(newConfig)
	assert.FileExists(t, filepath.Join(os.TempDir(), "testlog.log"))
}

func TestPlatformServer_StartServer(t *testing.T) {
	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, bus.GetBus(), func(t2 *testing.T) {
		rsp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		assert.Nil(t, err)

		_, err = ioutil.ReadAll(rsp.Body)
		assert.Nil(t, err)
		assert.Equal(t, 404, rsp.StatusCode)
		ps.StopServer()
		wg.Done()
	})

	wg.Wait()
}

func TestPlatformServer_RegisterService(t *testing.T) {
	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	err := ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	assert.Nil(t, err)
	service.GetServiceRegistry().UnregisterService(services.PingPongServiceChan)
}

func TestPlatformServer_SetHttpChannelBridge(t *testing.T) {
	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, bus.GetBus(), func(t2 *testing.T) {
		rsp, err := http.Get(fmt.Sprintf("http://localhost:%d/rest/ping-pong2?message=hello", port))
		assert.Nil(t, err)

		body, err := ioutil.ReadAll(rsp.Body)
		assert.Nil(t, err)
		assert.Contains(t, string(body), "hello")
		ps.StopServer()
		service.GetServiceRegistry().UnregisterService(services.PingPongServiceChan)
		wg.Done()
	})

	wg.Wait()
}

func TestPlatformServer_UnknownRequest(t *testing.T) {
	bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	defer service.GetServiceRegistry().UnregisterService(services.PingPongServiceChan)
	setupBridge(ps, "/ping", "GET", services.PingPongServiceChan, "bubble")

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, bus.GetBus(), func(t2 *testing.T) {
		rsp, err := http.Get(fmt.Sprintf("http://localhost:%d/ping?msg=hello", port))
		assert.Nil(t, err)

		body, err := ioutil.ReadAll(rsp.Body)
		assert.Nil(t, err)
		assert.Contains(t, string(body), "unsupported request")

		ps.StopServer()
		wg.Done()
	})

	wg.Wait()
}

func setupBridge(ps PlatformServer, endpoint, method, channel, request string) {
	bridgeConfig := &service.RESTBridgeConfig{
		ServiceChannel: channel,
		Uri:            endpoint,
		Method:         method,
		AllowHead:      false,
		AllowOptions:   false,
		FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
			q := r.URL.Query()
			return model.Request{
				Id:      &uuid.UUID{},
				Payload: q.Get("msg"),
				Request: request}

		},
	}
	ps.SetHttpChannelBridge(bridgeConfig)
}

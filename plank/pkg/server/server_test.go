// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/services"
	"github.com/vmware/transport-go/service"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	newConfig := GetBasicTestServerConfig(os.TempDir(), filepath.Join(os.TempDir(), "testlog.log"), "stdout", "stderr", port, true)
	ps := NewPlatformServer(newConfig)
	ps.(*platformServer).eventbus = newBus
	assert.FileExists(t, filepath.Join(os.TempDir(), "testlog.log"))
}

func TestPlatformServer_StartServer(t *testing.T) {
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	ps.(*platformServer).eventbus = newBus
	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(t2 *testing.T) {
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
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	ps.(*platformServer).eventbus = newBus

	err := ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	assert.Nil(t, err)
}

func TestPlatformServer_SetHttpPathPrefixChannelBridge(t *testing.T) {
	// get a new bus instance and create a new platform server instance
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	ps.(*platformServer).eventbus = newBus

	// register a service
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)

	// set PathPrefix bridge
	bridgeConfig := &service.RESTBridgeConfig{
		ServiceChannel: services.PingPongServiceChan,
		Uri:            "/ping-pong",
		FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
			return model.Request{
				Payload: "hello",
				Request: "ping-get",
			}
		},
	}
	ps.SetHttpPathPrefixChannelBridge(bridgeConfig)

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(t2 *testing.T) {
		// GET
		rsp, err := http.Get(fmt.Sprintf("http://localhost:%d/ping-pong", port))
		assert.Nil(t, err)

		body, err := ioutil.ReadAll(rsp.Body)
		assert.Nil(t, err)
		assert.Contains(t, string(body), "hello")

		// POST
		rsp, err = http.Post(fmt.Sprintf("http://localhost:%d/ping-pong", port), "application/json", strings.NewReader(""))
		assert.Nil(t, err)
		body, err = ioutil.ReadAll(rsp.Body)
		assert.Nil(t, err)
		assert.Contains(t, string(body), "hello")

		// DELETE
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:%d/ping-pong", port), strings.NewReader(""))
		rsp, err = ctxhttp.Do(context.Background(), http.DefaultClient, req)
		assert.Nil(t, err)
		body, err = ioutil.ReadAll(rsp.Body)
		assert.Nil(t, err)
		assert.Contains(t, string(body), "hello")

		ps.StopServer()
		service.GetServiceRegistry().UnregisterService(services.PingPongServiceChan)
		wg.Done()
	})

	wg.Wait()
}

func TestPlatformServer_SetHttpChannelBridge(t *testing.T) {
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	ps.(*platformServer).eventbus = newBus
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(t2 *testing.T) {
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
	newBus := bus.ResetBus()
	service.ResetServiceRegistry()
	port := GetTestPort()
	config := GetBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", port, true)
	ps := NewPlatformServer(config)
	ps.(*platformServer).eventbus = newBus
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	defer service.GetServiceRegistry().UnregisterService(services.PingPongServiceChan)
	setupBridge(ps, "/ping", "GET", services.PingPongServiceChan, "bubble")

	syschan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go ps.StartServer(syschan)
	RunWhenServerReady(t, newBus, func(t2 *testing.T) {
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

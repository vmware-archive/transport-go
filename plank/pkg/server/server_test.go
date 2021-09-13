// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
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
	"testing"
)

var config *PlatformServerConfig
var ps PlatformServer

func TestMain(m *testing.M) {
	config = getBasicTestServerConfig(os.TempDir(), "stdout", "stdout", "stderr", 9980, true)
	ps = NewPlatformServer(config)
	syschan := make(chan os.Signal, 1)
	go ps.StartServer(syschan)
	os.Exit(runTestMainWhenServerReady(m, bus.GetBus()))
}

func TestNewPlatformServer(t *testing.T) {
	assert.NotNil(t, ps)
}

func TestNewPlatformServer_EmptyRootDir(t *testing.T) {
	newConfig := getBasicTestServerConfig("", "stdout", "stdout", "stderr", 80, true)
	NewPlatformServer(newConfig)
	wd, _ := os.Getwd()
	assert.Equal(t, wd, newConfig.RootDir)
}

func TestNewPlatformServer_FileLog(t *testing.T) {
	defer func() {
		_ = os.Remove(filepath.Join(os.TempDir(), "testlog.log"))
	}()

	newConfig := getBasicTestServerConfig(os.TempDir(), filepath.Join(os.TempDir(), "testlog.log"), "stdout", "stderr", 80, true)
	NewPlatformServer(newConfig)
	assert.FileExists(t, filepath.Join(os.TempDir(), "testlog.log"))
}

func TestPlatformServer_StartServer(t *testing.T) {
	rsp, err := http.Get("http://localhost:9980")
	assert.Nil(t, err)

	_, err = ioutil.ReadAll(rsp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 404, rsp.StatusCode)

}

func TestPlatformServer_RegisterService(t *testing.T) {
	err := ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	assert.Nil(t, err)
}

func TestPlatformServer_SetHttpChannelBridge(t *testing.T) {
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	setupBridge(ps, "/pong", "GET", services.PingPongServiceChan, "ping-get")

	rsp, err := http.Get("http://localhost:9980/pong?msg=hello")
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(rsp.Body)
	assert.Nil(t, err)
	assert.Contains(t, string(body), "hello")
}

func TestPlatformServer_UnknownRequest(t *testing.T) {
	_ = ps.RegisterService(services.NewPingPongService(), services.PingPongServiceChan)
	setupBridge(ps, "/ping", "GET", services.PingPongServiceChan, "bubble")

	rsp, err := http.Get("http://localhost:9980/ping?msg=hello")
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(rsp.Body)
	assert.Nil(t, err)
	assert.Contains(t, string(body), "unsupported request")
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

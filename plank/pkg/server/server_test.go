package server

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jooskim/plank/services"
	"github.com/jooskim/plank/utils"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/model"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var config *PlatformServerConfig
var ps PlatformServer

func TestMain(m *testing.M) {
	config = &PlatformServerConfig{
		RootDir: "/tmp/",
		Host:    "localhost",
		Port:    9980,
		LogConfig: &utils.LogConfig{
			OutputLog:     "stdout",
			FormatOptions: &utils.LogFormatOption{},
		},
	}
	ps = NewPlatformServer(config)
	syschan := make(chan os.Signal, 1)
	go ps.StartServer(syschan)
	time.Sleep(2 * time.Second)

	os.Exit(m.Run())
}

func TestNewPlatformServer(t *testing.T) {
	assert.NotNil(t, ps)
}

func TestNewPlatformServer_EmptyRootDir(t *testing.T) {
	newConfig := &PlatformServerConfig{
		Host: "localhost",
		Port: 80,
		LogConfig: &utils.LogConfig{
			OutputLog:     "stdout",
			FormatOptions: &utils.LogFormatOption{},
		},
	}
	NewPlatformServer(newConfig)
	assert.Contains(t, newConfig.RootDir, fmt.Sprintf("pkg%cserver", filepath.Separator))
}

func TestNewPlatformServer_FileLog(t *testing.T) {
	defer func() {
		_ = os.Remove("/tmp/testlog.log")
	}()

	newConfig := &PlatformServerConfig{
		RootDir: "/tmp/",
		Host:    "localhost",
		Port:    80,
		LogConfig: &utils.LogConfig{
			OutputLog:     "testlog.log",
			FormatOptions: &utils.LogFormatOption{},
		},
	}
	NewPlatformServer(newConfig)
	assert.FileExists(t, "/tmp/testlog.log")
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
	setupBridge(ps, "/pong", "GET", services.PingPongServiceChan, "ping2")

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
	ps.SetHttpChannelBridge(endpoint, method, channel, func(w http.ResponseWriter, r *http.Request) model.Request {
		q := r.URL.Query()
		return model.Request{
			Id:      &uuid.UUID{},
			Payload: q.Get("msg"),
			Request: request}

	})
}

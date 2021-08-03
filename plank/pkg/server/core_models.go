package server

import (
	"crypto/tls"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/plank/pkg/middleware"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/service"
	"github.com/vmware/transport-go/stompserver"
	"net/http"
	"os"
	"sync"
	"time"
)

type PlatformServerConfig struct {
	RootDir                  string              `json:"root_dir"`
	StaticDir                []string            `json:"static_dir"`
	SpaConfig                *SpaConfig          `json:"spa_config"`
	Host                     string              `json:"host"`
	Port                     int                 `json:"port"`
	LogConfig                *utils.LogConfig    `json:"log_config"`
	FabricConfig             *FabricBrokerConfig `json:"fabric_config"`
	TLSCertConfig            *TLSCertConfig      `json:"tls_config"`
	EnablePrometheus         bool                `json:"enable_prometheus"`
	Debug                    bool                `json:"debug"`
	NoBanner                 bool                `json:"no_banner"`
	ShutdownTimeoutInMinutes time.Duration       `json:"shutdown_timeout_in_minutes"`
}

type TLSCertConfig struct {
	CertFile                  string `json:"cert_file"`
	KeyFile                   string `json:"key_file"`
	SkipCertificateValidation bool   `json:"skip_certificate_validation"`
}

type FabricBrokerConfig struct {
	FabricEndpoint string              `json:"fabric_endpoint"`
	EndpointConfig *bus.EndpointConfig `json:"endpoint_config"`
}

type PlatformServer interface {
	StartServer(syschan chan os.Signal)
	StopServer()
	RegisterService(svc service.FabricService, svcChannel string) error
	SetHttpChannelBridge(bridgeConfig *service.RESTBridgeConfig)
	SetStaticRoute(prefix, fullpath string)
	CustomizeTLSConfig(tls *tls.Config) error
	GetRestBridgeSubRoute(uri, method string) (*mux.Route, error)
	GetMiddlewareManager() middleware.MiddlewareManager
}

type platformServer struct {
	HttpServer                   *http.Server
	SyscallChan                  chan os.Signal
	serverConfig                 *PlatformServerConfig
	middlewareManager            middleware.MiddlewareManager
	router                       *mux.Router
	out                          *os.File
	endpointHandlerMap           map[string]http.HandlerFunc
	serviceChanToBridgeEndpoints map[string][]string
	fabricConn                   stompserver.RawConnectionListener
	serverAvailability           *serverAvailability
	lock                         sync.Mutex
}

type transportChannelResponse struct {
	message *model.Message
	err     error
}

type serverAvailability struct {
	http   bool
	fabric bool
}

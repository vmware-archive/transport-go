// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"crypto/tls"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/pkg/middleware"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"github.com/vmware/transport-go/stompserver"
	"net/http"
	"os"
	"sync"
	"time"
)

// PlatformServerConfig holds all the core configuration needed for the functionality of Plank
type PlatformServerConfig struct {
	RootDir           string              `json:"root_dir"`                       // root directory the server should base itself on
	StaticDir         []string            `json:"static_dir"`                     // static content folders that HTTP server should serve
	SpaConfig         *SpaConfig          `json:"spa_config"`                     // single page application configuration
	Host              string              `json:"host"`                           // hostname for the server
	Port              int                 `json:"port"`                           // port for the server
	LogConfig         *utils.LogConfig    `json:"log_config"`                     // log configuration (plank, Http access and error logs)
	FabricConfig      *FabricBrokerConfig `json:"fabric_config"`                  // Fabric (websocket) configuration
	TLSCertConfig     *TLSCertConfig      `json:"tls_config"`                     // TLS certificate configuration
	EnablePrometheus  bool                `json:"enable_prometheus"`              // whether to enable Prometheus for runtime metrics
	Debug             bool                `json:"debug"`                          // enable debug logging
	NoBanner          bool                `json:"no_banner"`                      // start server without displaying the banner
	ShutdownTimeout   time.Duration       `json:"shutdown_timeout_in_minutes"`    // graceful server shutdown timeout in minutes
	RestBridgeTimeout time.Duration       `json:"rest_bridge_timeout_in_minutes"` // rest bridge timeout in minutes
}

// TLSCertConfig wraps around key information for TLS configuration
type TLSCertConfig struct {
	CertFile                  string `json:"cert_file"`                   // path to certificate file
	KeyFile                   string `json:"key_file"`                    // path to private key file
	SkipCertificateValidation bool   `json:"skip_certificate_validation"` // whether to skip certificate validation (useful for self-signed cert)
}

// FabricBrokerConfig defines the endpoint for WebSocket as well as detailed endpoint configuration
type FabricBrokerConfig struct {
	FabricEndpoint string              `json:"fabric_endpoint"` // URI to WebSocket endpoint
	EndpointConfig *bus.EndpointConfig `json:"endpoint_config"` // STOMP configuration
}

// PlatformServer exposes public API methods that control the behavior of the Plank instance.
type PlatformServer interface {
	StartServer(syschan chan os.Signal)                                         // start server
	StopServer()                                                                // stop server
	RegisterService(svc service.FabricService, svcChannel string) error         // register a new service at given channel
	SetHttpChannelBridge(bridgeConfig *service.RESTBridgeConfig)                // set up a REST bridge for a service
	SetStaticRoute(prefix, fullpath string, middlewareFn ...mux.MiddlewareFunc) // set up a static content route
	SetHttpPathPrefixChannelBridge(bridgeConfig *service.RESTBridgeConfig)      // set up a REST bridge for a path prefix for a service.
	CustomizeTLSConfig(tls *tls.Config) error                                   // used to replace default tls.Config for HTTP server with a custom config
	GetRestBridgeSubRoute(uri, method string) (*mux.Route, error)               // get *mux.Route that maps to the provided uri and method
	GetMiddlewareManager() middleware.MiddlewareManager                         // get middleware manager

}

// platformServer is the main struct that holds all components together including servers, various managers etc.
type platformServer struct {
	HttpServer                   *http.Server                      // Http server instance
	SyscallChan                  chan os.Signal                    // syscall channel to receive SIGINT, SIGKILL events
	serverConfig                 *PlatformServerConfig             // server config instance
	middlewareManager            middleware.MiddlewareManager      // middleware maanger instance
	router                       *mux.Router                       // *mux.Router instance
	routerConcurrencyProtection  *int32                            // atomic int32 to protect the main router being concurrently written to
	out                          *os.File                          // platform log output pointer
	endpointHandlerMap           map[string]http.HandlerFunc       // internal map to store rest endpoint -handler mappings
	serviceChanToBridgeEndpoints map[string][]string               // internal map to store service channel - endpoint handler key mappings
	fabricConn                   stompserver.RawConnectionListener // WebSocket listener instance
	ServerAvailability           *ServerAvailability               // server availability (not much used other than for internal monitoring for now)
	lock                         sync.Mutex                        // lock
	messageBridgeMap             map[string]*MessageBridge
}

// MessageBridge is a conduit used for returning service responses as HTTP responses
type MessageBridge struct {
	ServiceListenStream bus.MessageHandler  // message handler returned by bus.ListenStream responsible for relaying back messages as HTTP responses
	payloadChannel      chan *model.Message // internal golang channel used for passing bus responses/errors across goroutines
}

// ServerAvailability contains boolean fields to indicate what components of the system are available or not
type ServerAvailability struct {
	Http   bool // Http server availability
	Fabric bool // stomp broker availability
}

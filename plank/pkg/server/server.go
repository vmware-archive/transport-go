// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/plank/pkg/middleware"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"github.com/vmware/transport-go/stompserver"
)

// NewPlatformServer configures and returns a new platformServer instance
func NewPlatformServer(config *PlatformServerConfig) PlatformServer {
	ps := new(platformServer)
	sanitizeConfigRootPath(config)
	ps.serverConfig = config
	ps.serverAvailability = &serverAvailability{}
	ps.routerConcurrencyProtection = new(int32)
	ps.initialize()

	return ps
}

// NewPlatformServerFromConfig returns a new instance of PlatformServer based on the config JSON file provided as configPath
func NewPlatformServerFromConfig(configPath string) (PlatformServer, error) {
	var config PlatformServerConfig

	// no config no server
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// malformed config no server as well
	if err = json.Unmarshal(configBytes, &config); err != nil {
		return nil, err
	}

	ps := new(platformServer)
	sanitizeConfigRootPath(&config)

	// ensure references to file system paths are relative to config.RootDir
	config.LogConfig.OutputLog = utils.JoinBasePathIfRelativeRegularFilePath(config.LogConfig.Root, config.LogConfig.OutputLog)
	config.LogConfig.AccessLog = utils.JoinBasePathIfRelativeRegularFilePath(config.LogConfig.Root, config.LogConfig.AccessLog)
	config.LogConfig.ErrorLog = utils.JoinBasePathIfRelativeRegularFilePath(config.LogConfig.Root, config.LogConfig.ErrorLog)

	// handle invalid duration by setting it to the default value of 1 minute
	if config.RestBridgeTimeoutInMinutes <= 0 {
		config.RestBridgeTimeoutInMinutes = 1
	}

	// the raw value from the config.json needs to be multiplied by time.Minute otherwise it's interpreted as nanosecond
	config.RestBridgeTimeoutInMinutes = config.RestBridgeTimeoutInMinutes * time.Minute

	if config.TLSCertConfig != nil {
		if !path.IsAbs(config.TLSCertConfig.CertFile) {
			config.TLSCertConfig.CertFile = path.Clean(path.Join(config.RootDir, config.TLSCertConfig.CertFile))
		}

		if !path.IsAbs(config.TLSCertConfig.KeyFile) {
			config.TLSCertConfig.KeyFile = path.Clean(path.Join(config.RootDir, config.TLSCertConfig.KeyFile))
		}
	}

	ps.serverConfig = &config
	ps.serverAvailability = &serverAvailability{}
	ps.routerConcurrencyProtection = new(int32)
	ps.initialize()
	return ps, nil
}

// CreateServerConfig creates a new instance of PlatformServerConfig and returns the pointer to it.
func CreateServerConfig() (*PlatformServerConfig, error) {
	factory := &serverConfigFactory{}
	factory.configureFlags(pflag.CommandLine)
	factory.parseFlags()
	return generatePlatformServerConfig(factory)
}

// CreateServerConfigForCobraCommand performs the same as CreateServerConfig but loads the flags to
// the provided cobra Command's *pflag.FlagSet instead of the global FlagSet instance.
func CreateServerConfigForCobraCommand(cmd *cobra.Command) (*PlatformServerConfig, error) {
	factory := &serverConfigFactory{}
	factory.configureFlags(cmd.Flags())
	factory.parseFlags()
	return generatePlatformServerConfig(factory)
}

// StartServer starts listening on the host and port as specified by ServerConfig
func (ps *platformServer) StartServer(syschan chan os.Signal) {
	connClosed := make(chan struct{})

	ps.SyscallChan = syschan
	signal.Notify(ps.SyscallChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// ensure port is available
	ps.checkPortAvailability()

	// get http access log and error log file pointers ready

	// finalize handler by setting out writer
	ps.loadGlobalHttpHandler(ps.router)

	// if path for SPA app is provided set it up
	// NOTE: the reason SPA app route is configured after service & static routes are configured is that sometimes you may want the UI
	// to be served at the root (/) route in which case setting up an SPA route first will basically mask all other URIs.
	// TODO: error if the base uri conflicts with another URI registered before
	if ps.serverConfig.SpaConfig != nil {
		for _, asset := range ps.serverConfig.SpaConfig.StaticAssets {
			folderPath, uri := utils.DeriveStaticURIFromPath(asset)
			ps.SetStaticRoute(utils.SanitizeUrl(uri, false), folderPath)
		}

		ps.router.PathPrefix(ps.serverConfig.SpaConfig.BaseUri).Name("spa-base").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resource := "index.html"

			// if the URI contains an extension we treat it as access to static resources
			if len(filepath.Ext(r.URL.Path)) > 0 {
				resource = filepath.Clean(r.URL.Path)
			}
			http.ServeFile(w, r, filepath.Join(ps.serverConfig.SpaConfig.RootFolder, resource))
		})
	}

	go func() {
		ps.serverAvailability.http = true
		if ps.serverConfig.TLSCertConfig != nil {
			utils.Log.Infof("Starting HTTP server at %s:%d with TLS", ps.serverConfig.Host, ps.serverConfig.Port)
			_ = ps.HttpServer.ListenAndServeTLS(ps.serverConfig.TLSCertConfig.CertFile, ps.serverConfig.TLSCertConfig.KeyFile)
		} else {
			utils.Log.Infof("Starting HTTP server at %s:%d", ps.serverConfig.Host, ps.serverConfig.Port)
			_ = ps.HttpServer.ListenAndServe()
		}
	}()

	// if fabric broker configuration is found, start the broker
	if ps.serverConfig.FabricConfig != nil {
		var err error
		utils.Log.Infof("Starting Fabric broker at %s:%d%s",
			ps.serverConfig.Host, ps.serverConfig.Port, ps.serverConfig.FabricConfig.FabricEndpoint)

		// TODO: consider tightening access by allowing configuring allowedOrigins
		ps.fabricConn, err = stompserver.NewWebSocketConnectionFromExistingHttpServer(
			ps.HttpServer,
			ps.router,
			ps.serverConfig.FabricConfig.FabricEndpoint,
			nil)

		// if creation of listener fails, crash and burn
		if err != nil {
			panic(err)
		}

		// otherwise, start the broker
		go func() {
			ps.serverAvailability.fabric = true
			if err := bus.GetBus().StartFabricEndpoint(ps.fabricConn, *ps.serverConfig.FabricConfig.EndpointConfig); err != nil {
				panic(err)
			}
		}()
	}

	// spawn another goroutine to respond to syscall to shut down servers and terminate the main thread
	go func() {
		<-ps.SyscallChan
		ps.StopServer()
		close(connClosed)
	}()

	<-connClosed
}

// StopServer attempts to gracefully stop the HTTP and STOMP server if running
func (ps *platformServer) StopServer() {
	utils.Log.Infoln("Server shutting down")
	ps.serverAvailability.http = false

	baseCtx := context.Background()
	shutdownCtx, cancel := context.WithTimeout(baseCtx, ps.serverConfig.ShutdownTimeoutInMinutes*time.Minute)

	go func() {
		select {
		case <-shutdownCtx.Done():
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				utils.Log.Fatalf(
					"Server failed to gracefully shut down after %s",
					(ps.serverConfig.ShutdownTimeoutInMinutes * time.Minute).String())
			}
		}
	}()
	defer cancel()

	// call all registered services' OnServerShutdown() hook
	svcRegistry := service.GetServiceRegistry()
	lcm := service.GetServiceLifecycleManager()
	wg := sync.WaitGroup{}
	for _, svcChannel := range svcRegistry.GetAllServiceChannels() {
		hooks := lcm.GetServiceHooks(svcChannel)
		if hooks != nil {
			utils.Log.Infof("Teardown in progress for service at '%s'", svcChannel)
			wg.Add(1)
			go func(cName string, h service.ServiceLifecycleHookEnabled) {
				h.OnServerShutdown()
				utils.Log.Infof("Teardown completed for service at '%s'", cName)
				wg.Done()

			}(svcChannel, hooks)
		}
	}

	// start graceful shutdown
	err := ps.HttpServer.Shutdown(shutdownCtx)
	if err != nil {
		utils.Log.Errorln(err)
	}

	if ps.fabricConn != nil {
		err = ps.fabricConn.Close()
		if err != nil {
			utils.Log.Errorln(err)
		}
		ps.serverAvailability.fabric = false
	}

	// wait for all teardown jobs to be done. if shutdown deadline arrives earlier
	// the main thread will be terminated forcefully
	wg.Wait()
}

// SetStaticRoute adds a route where static resources will be served
func (ps *platformServer) SetStaticRoute(prefix, fullpath string) {
	ps.router.Handle(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, prefix+"/", http.StatusMovedPermanently)
	}))

	ndir := NoDirFileSystem{http.Dir(fullpath)}
	ps.router.PathPrefix(prefix + "/").Name(fmt.Sprintf("static-%s", prefix)).Handler(http.StripPrefix(prefix,
		middleware.BasicSecurityHeaderMiddleware.Intercept(
			middleware.NewCacheControlWrapperMiddleware(time.Hour*0).Intercept(http.FileServer(ndir)))))
}

// RegisterService registers a fabric service with Bifrost
func (ps *platformServer) RegisterService(svc service.FabricService, svcChannel string) error {
	sr := service.GetServiceRegistry()
	err := sr.RegisterService(svc, svcChannel)
	svcType := reflect.TypeOf(svc)

	if err == nil {
		utils.Log.Infof("Service '%s' registered at channel '%s'", svcType.String(), svcChannel)
		svcLifecycleManager := service.GetServiceLifecycleManager()
		var hooks service.ServiceLifecycleHookEnabled
		if hooks = svcLifecycleManager.GetServiceHooks(svcChannel); hooks == nil {
			// if service has no lifecycle hooks mark the channel as ready straight up
			storeManager := bus.GetBus().GetStoreManager()
			store := storeManager.GetStore(service.ServiceReadyStore)
			store.Put(svcChannel, true, service.ServiceInitStateChange)
			utils.Log.Infof("Service '%s' initialized successfully", svcType.String())
		}
	}
	return err
}

// SetHttpChannelBridge establishes a conduit between the the transport service channel and an HTTP endpoint
// that allows a client to invoke the service via REST.
func (ps *platformServer) SetHttpChannelBridge(bridgeConfig *service.RESTBridgeConfig) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	endpointHandlerKey := bridgeConfig.Uri + "-" + bridgeConfig.Method

	if _, ok := ps.endpointHandlerMap[endpointHandlerKey]; ok {
		utils.Log.Warnf("Endpoint '%s (%s)' is already associated with a handler. "+
			"Try another endpoint or remove it before assigning a new handler", bridgeConfig.Uri, bridgeConfig.Method)
		return
	}

	// create a map for service channel - bridges mapping if it does not exist
	if ps.serviceChanToBridgeEndpoints[bridgeConfig.ServiceChannel] == nil {
		ps.serviceChanToBridgeEndpoints[bridgeConfig.ServiceChannel] = make([]string, 0)
	}

	// build endpoint handler
	ps.endpointHandlerMap[endpointHandlerKey] = buildEndpointHandler(
		bridgeConfig.ServiceChannel,
		bridgeConfig.FabricRequestBuilder,
		ps.serverConfig.RestBridgeTimeoutInMinutes)

	ps.serviceChanToBridgeEndpoints[bridgeConfig.ServiceChannel] = append(
		ps.serviceChanToBridgeEndpoints[bridgeConfig.ServiceChannel], endpointHandlerKey)

	permittedMethods := []string{bridgeConfig.Method}
	if bridgeConfig.AllowHead {
		permittedMethods = append(permittedMethods, http.MethodHead)
	}
	if bridgeConfig.AllowOptions {
		permittedMethods = append(permittedMethods, http.MethodOptions)
	}

	// NOTE: mux.Router does not have mutex or any locking mechanism so it could sometimes lead to concurrency write
	// panics. following is to ensure the modification to ps.router can happen only once per thread
	for !atomic.CompareAndSwapInt32(ps.routerConcurrencyProtection, 0, 1) {
		time.Sleep(1 * time.Nanosecond)
	}

	ps.router.
		Path(bridgeConfig.Uri).
		Methods(permittedMethods...).
		Name(fmt.Sprintf("bridge-%s-%s", bridgeConfig.Uri, bridgeConfig.Method)).
		Handler(ps.endpointHandlerMap[endpointHandlerKey])
	if !atomic.CompareAndSwapInt32(ps.routerConcurrencyProtection, 1, 0) {
		panic("Concurrency write detected!")
	}

	utils.Log.Infof(
		"Service channel '%s' is now bridged to a REST endpoint %s (%s)",
		bridgeConfig.ServiceChannel, bridgeConfig.Uri, bridgeConfig.Method)
}

// GetMiddlewareManager returns the MiddleManager instance
func (ps *platformServer) GetMiddlewareManager() middleware.MiddlewareManager {
	return ps.middlewareManager
}

func (ps *platformServer) GetRestBridgeSubRoute(uri, method string) (*mux.Route, error) {
	route, err := ps.getSubRoute(fmt.Sprintf("bridge-%s-%s", uri, method))
	if route == nil {
		return nil, fmt.Errorf("no route exists at %s (%s) exists", uri, method)
	}
	return route, err
}

// CustomizeTLSConfig is used to create a customized TLS configuration for use with http.Server.
// this function needs to be called before the server starts, otherwise it will error out.
func (c *platformServer) CustomizeTLSConfig(tls *tls.Config) error {
	if c.serverAvailability.http || c.serverAvailability.fabric {
		return fmt.Errorf("TLS configuration can be provided only if the server is not running")
	}
	c.HttpServer.TLSConfig = tls
	return nil
}

// clearHttpChannelBridgesForService takes serviceChannel, gets all mux.Route instances associated with
// the service and removes them while keeping the rest of the routes intact. returns the pointer
// of a new instance of mux.Router.
func (ps *platformServer) clearHttpChannelBridgesForService(serviceChannel string) *mux.Router {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	// NOTE: gorilla mux doesn't allow us to mutate routes field of the Router struct which is critical in rerouting incoming
	// requests to the new route. there is not a public API that allows us to do it so we're instead creating a new instance of
	// Router and assigning the existing config and route. this means `ps.route` is treated as immutable and will be
	// replaced with a new instance of mux.Router by the operation performed in this function

	// walk over existing routes and store them temporarily EXCEPT the ones that are being overwritten which can
	// be tracked by the service channel
	newRouter := mux.NewRouter().Schemes("http", "https").Subrouter()
	lookupMap := make(map[string]bool)
	for _, key := range ps.serviceChanToBridgeEndpoints[serviceChannel] {
		lookupMap["bridge-"+key] = true
	}

	ps.router.Walk(func(r *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		name := r.GetName()
		path, _ := r.GetPathTemplate()
		handler := r.GetHandler()
		methods, _ := r.GetMethods()
		// do not want to copy over the routes that will be overridden
		if lookupMap[name] {
			utils.Log.Debugf("route '%s' will be overridden so not copying over to the new router instance", name)
		} else {
			newRouter.Name(name).Path(path).Methods(methods...).Handler(handler)
		}
		return nil
	})

	// if in override mode delete existing mappings associated with the service
	existingMappings := ps.serviceChanToBridgeEndpoints[serviceChannel]
	ps.serviceChanToBridgeEndpoints[serviceChannel] = make([]string, 0)
	for _, handlerKey := range existingMappings {
		utils.Log.Infof("Removing existing service - REST mapping '%s' for service '%s'", handlerKey, serviceChannel)
		delete(ps.endpointHandlerMap, handlerKey)
	}
	return newRouter
}

func (ps *platformServer) getSubRoute(name string) (*mux.Route, error) {
	route := ps.router.Get(name)
	if route == nil {
		return nil, fmt.Errorf("no route exists under name %s", name)
	}
	return route, nil
}

func (ps *platformServer) loadGlobalHttpHandler(h *mux.Router) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.router = h
	ps.HttpServer.Handler = handlers.RecoveryHandler()(
		handlers.CompressHandler(
			handlers.ProxyHeaders(
				handlers.CombinedLoggingHandler(
					ps.serverConfig.LogConfig.GetAccessLogFilePointer(), ps.router))))
}

func (ps *platformServer) configureRobotsPath() {
	ps.router.Path("/robots.txt").Name("robots").Handler(
		middleware.BasicSecurityHeaderMiddleware.Intercept(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wd, _ := os.Getwd()
				robotsFilePath := filepath.Join(wd, "robots.txt")
				utils.Log.Debugf("Attempting to load robots.txt from %s", robotsFilePath)

				// if robots.txt was not found or cannot be loaded for some reason, ignore it and return 404
				// still, return the detailed error as a debug message to facilitate ease of troubleshooting
				if _, err := os.Stat(robotsFilePath); err != nil {
					utils.Log.Debugln(err)
					http.NotFound(w, r)
					return
				}

				b, err := ioutil.ReadFile(robotsFilePath)
				if err != nil {
					utils.Log.Debugln(err)
					http.NotFound(w, r)
					return
				}

				_, _ = w.Write(b)
			})))
}

func (ps *platformServer) checkPortAvailability() {
	// is the port free?
	_, err := net.Dial("tcp", fmt.Sprintf(":%d", ps.serverConfig.Port))

	// connection should fail otherwise it means there's already a listener on the host+port combination, in which case we stop here
	if err == nil {
		utils.Log.Fatalf("Server could not start at %s:%d because another process is using it. Please try another endpoint.",
			ps.serverConfig.Host, ps.serverConfig.Port)
	}
}

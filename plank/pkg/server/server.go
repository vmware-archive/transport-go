// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
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

func CreateServerConfigFromUrfaveCLIContext(c *cli.Context) (*PlatformServerConfig, error) {
	return generatePlatformServerConfig(c)
}

func CreateServerConfig() (*PlatformServerConfig, error) {
	factory := &serverConfigFactory{}
	factory.configureFlags()
	factory.parseFlags()
	return generatePlatformServerConfig(factory)
}

// StartServer starts listening on the host and port as specified by ServerConfig
func (ps *platformServer) StartServer(syschan chan os.Signal) {
	connClosed := make(chan struct{})

	// start bifrost core and service registry
	bus.GetBus()
	service.GetServiceRegistry()

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
			http.ServeFile(w, r, filepath.Join(ps.serverConfig.SpaConfig.RootFolder, "index.html"))
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

	ps.endpointHandlerMap[endpointHandlerKey] = buildEndpointHandler(bridgeConfig.ServiceChannel, bridgeConfig.FabricRequestBuilder)
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
		"Service channel '%s' is now bridged to a REST endpoint %s (%s)\n",
		bridgeConfig.ServiceChannel, bridgeConfig.Uri, bridgeConfig.Method)
}

func (ps *platformServer) clearHttpChannelBridgesForService(serviceChannel string) *mux.Router {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	// NOTE: gorilla mux doesn't allow us to mutate routes field of the Router struct which is critical in rerouting incoming
	// requests to the new route. there is not a public API that allows us to do it so we're instead creating a new instance of
	// Router and assigning the existing config and route

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
			handlers.CombinedLoggingHandler(
				ps.serverConfig.LogConfig.GetAccessLogFilePointer(), ps.router)))
}

func buildEndpointHandler(svcChannel string, reqBuilder func(w http.ResponseWriter, r *http.Request) model.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Errorln(r)
				http.Error(w, "Internal Server Error", 500)
			}
		}()

		// set context that would expire after 30 seconds by default to prevent requests from hanging forever
		ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelFn()
		h, err := bus.GetBus().ListenOnce(svcChannel)
		if err != nil {
			panic(err)
		}

		// set up a channel through which to receive the raw response from transport channel
		// handler function runs in another thread so we need to utilize channel to use the correct writer.
		chanReturn := make(chan *transportChannelResponse)
		h.Handle(func(message *model.Message) {
			chanReturn <- &transportChannelResponse{message: message}
		}, func(err error) {
			chanReturn <- &transportChannelResponse{err: err}
		})

		// relay the request to transport channel
		reqModel := reqBuilder(w, r)
		err = bus.GetBus().SendRequestMessage(svcChannel, reqModel, reqModel.Id)

		// get a response from the channel, render the results using ResponseWriter and log the data/error
		// to the console as well.
		select {
		case <-ctx.Done():
			http.Error(w, "Request timed out", 500)
		case chanResponse := <-chanReturn:
			if chanResponse.err != nil {
				utils.Log.WithError(chanResponse.err).Errorf(
					"Error received from channel %s:", svcChannel)
				http.Error(w, chanResponse.err.Error(), 500)
			} else {
				// only send the actual user payload not wrapper information
				response := chanResponse.message.Payload.(*model.Response)
				var respBody interface{}
				if response.Error {
					if response.Payload != nil {
						respBody = response.Payload
					} else {
						respBody = response
					}
				} else {
					respBody = response.Payload
				}

				utils.Log.WithFields(logrus.Fields{
					//"payload": respBody, // don't show this, we may be sending around big byte arrays
				}).Debugf("Response received from channel %s:", svcChannel)

				// if our message is an error and it has a code, lets send that back to the client.
				if response.Error {

					// we have to set the headers for the error response
					for k, v := range response.Headers {
						w.Header().Set(k, v)
					}

					// deal with the response body now, if set.
					n, e := json.Marshal(respBody)
					if e != nil {

						w.WriteHeader(response.ErrorCode)
						w.Write([]byte(response.ErrorMessage))
						return

					} else {
						w.WriteHeader(response.ErrorCode)
						w.Write(n)
						return
					}
				} else {
					// if the response has headers, set those headers. particularly if you're sending around
					// byte array data for things like zip files etc.
					for k, v := range response.Headers {
						w.Header().Set(k, v)
						if strings.ToLower(k) == "content-type" {
							respBody, err = utils.ConvertInterfaceToByteArray(v, respBody)
						}
					}

					var respBodyBytes []byte
					// ensure respBody is properly converted to a byte array as Content-Type header might not be
					// set in the request and the restBody could be in a format that can be json marshalled.
					respBodyBytes, err = marshalResponseBody(respBody)

					// write the non-error payload back.
					if _, err = w.Write(respBodyBytes); err != nil {
						utils.Log.WithError(err).Errorf("Error received from channel %s:", svcChannel)
						http.Error(w, err.Error(), 500)
					}
				}
			}
		}
	}
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

// printBanner prints out banner as well as brief server config
func (ps *platformServer) printBanner() {
	fmt.Println(` ______ __      ______  __   __  __  __    
/\  == /\ \    /\  __ \/\ "-.\ \/\ \/ /    
\ \  _-\ \ \___\ \  __ \ \ \-.  \ \  _"-.  
 \ \_\  \ \_____\ \_\ \_\ \_\\"\_\ \_\ \_\ 
  \/_/   \/_____/\/_/\/_/\/_/ \/_/\/_/\/_/ 
                                           `)
	utils.Infof("Host\t\t\t")
	fmt.Println(ps.serverConfig.Host)
	utils.Infof("Port\t\t\t")
	fmt.Println(ps.serverConfig.Port)

	if ps.serverConfig.FabricConfig != nil {
		utils.Infof("Fabric endpoint\t\t")
		fmt.Println(ps.serverConfig.FabricConfig.FabricEndpoint)
	}

	if len(ps.serverConfig.StaticDir) > 0 {
		utils.Infof("Static endpoints\t")
		for i, dir := range ps.serverConfig.StaticDir {
			_, p := utils.DeriveStaticURIFromPath(dir)
			fmt.Print(p)
			if i < len(ps.serverConfig.StaticDir)-1 {
				fmt.Print(", ")
			} else {
				fmt.Print("\n")
			}
		}
	}

	if ps.serverConfig.SpaConfig != nil {
		utils.Infof("SPA endpoint\t\t")
		fmt.Println(ps.serverConfig.SpaConfig.BaseUri)
		utils.Infof("SPA static assets\t")
		if len(ps.serverConfig.SpaConfig.StaticAssets) == 0 {
			fmt.Print("-")
		}
		for idx, asset := range ps.serverConfig.SpaConfig.StaticAssets {
			_, uri := utils.DeriveStaticURIFromPath(asset)
			fmt.Print(utils.SanitizeUrl(uri, false))
			if idx < len(ps.serverConfig.SpaConfig.StaticAssets)-1 {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}

	utils.Infof("Health endpoint\t\t")
	fmt.Println("/health")

	if ps.serverConfig.EnablePrometheus {
		utils.Infof("Prometheus endpoint\t")
		fmt.Println("/prometheus")
	}

	fmt.Println()

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

// initialize sets up basic configurations according to the serverConfig object such as setting output writer,
// log formatter, creating a router instance, and setting up an HttpServer instance.
func (ps *platformServer) initialize() {
	var err error

	// initialize service registry
	service.GetServiceRegistry()

	// initialize HTTP endpoint handlers map
	ps.endpointHandlerMap = map[string]http.HandlerFunc{}
	ps.serviceChanToBridgeEndpoints = make(map[string][]string, 0)

	// initialize log output streams
	if err = ps.serverConfig.LogConfig.PrepareLogFiles(); err != nil {
		panic(err)
	}

	// alias outputLogFp as ps.out for platform log outputs
	ps.out = ps.serverConfig.LogConfig.GetPlatformLogFilePointer()

	// set logrus out writer options and assign output stream to ps.out
	formatter := utils.CreateTextFormatterFromFormatOptions(ps.serverConfig.LogConfig.FormatOptions)
	utils.Log.SetFormatter(formatter)
	utils.Log.SetOutput(ps.out)

	// if debug flag is provided enable extra logging
	if ps.serverConfig.Debug {
		utils.Log.SetLevel(logrus.DebugLevel)
		utils.Log.Debugln("Debug logging enabled")
	}

	// set a new route handler
	ps.router = mux.NewRouter().Schemes("http", "https").Subrouter()

	// register a reserved path /robots.txt for setting crawler policies
	ps.configureRobotsPath()

	// register a reserved path /health for use with container orchestration layer like k8s
	ps.router.Path("/health").Name("health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// register a reserved path /prometheus for runtime metrics, if enabled
	if ps.serverConfig.EnablePrometheus {
		ps.router.Path("/prometheus").Name("prometheus").Methods(http.MethodGet).Handler(
			middleware.BasicSecurityHeaderMiddleware.Intercept(promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{
					EnableOpenMetrics: true,
				})))
	}

	// register static paths
	for _, dir := range ps.serverConfig.StaticDir {
		p, uri := utils.DeriveStaticURIFromPath(dir)
		utils.Log.Debugf("Serving static path %s at %s", p, uri)
		ps.SetStaticRoute(uri, p)
	}

	// create an http server instance
	ps.HttpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", ps.serverConfig.Port),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		ErrorLog:     log.New(ps.serverConfig.LogConfig.GetErrorLogFilePointer(), "ERROR ", log.LstdFlags),
	}

	// set up a listener to receive REST bridge configs for services and set them up according to their specs
	lcmChanHandler, err := bus.GetBus().ListenStreamForDestination(service.LifecycleManagerChannelName, bus.GetBus().GetId())
	if err != nil {
		utils.Log.Fatalln(err)
	}

	lcmChanHandler.Handle(func(message *model.Message) {
		request, ok := message.Payload.(*service.SetupRESTBridgeRequest)
		if !ok {
			utils.Log.Errorf("failed to set up REST bridge ")
		}

		// REST bridge setup done. now wait for service to be ready
		fabricSvc, _ := service.GetServiceRegistry().GetService(request.ServiceChannel)
		lcm := service.GetServiceLifecycleManager()
		svcReadyStore := bus.GetBus().GetStoreManager().GetStore(service.ServiceReadyStore)
		hooks := lcm.GetServiceHooks(request.ServiceChannel)

		if val, found := svcReadyStore.Get(request.ServiceChannel); !found || !val.(bool) {
			readyChan := hooks.OnServiceReady()
			svcReadyStore.Put(request.ServiceChannel, <-readyChan, service.ServiceInitStateChange)
			utils.Log.Infof("Service '%s' initialized successfully", reflect.TypeOf(fabricSvc).String())
			close(readyChan)
		}

		if request.Override {
			// clear old bridges affected by this override. there's a suboptimal workaround for mux.Router not
			// supporting a way to dynamically remove routers slice. see clearHttpChannelBridgesForService for details
			newRouter := ps.clearHttpChannelBridgesForService(request.ServiceChannel)
			ps.loadGlobalHttpHandler(newRouter)
		}

		for _, config := range request.Config {
			ps.SetHttpChannelBridge(config)
		}

	}, func(err error) {
		utils.Log.Errorln(err)
	})

	// instantiate a new middleware manager
	ps.middlewareManager = middleware.NewMiddlewareManager(&ps.endpointHandlerMap)

	// print out the quick summary of the server configuration, if NoBanner is false
	if !ps.serverConfig.NoBanner {
		ps.printBanner()
	}
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

// helpers

func generatePlatformServerConfig(i interface{}) (*PlatformServerConfig, error) {
	configFile := extractFlagValueFromProvider(i, "ConfigFile", "string").(string)
	host := extractFlagValueFromProvider(i, "Hostname", "string").(string)
	port := extractFlagValueFromProvider(i, "Port", "int").(int)
	rootDir := extractFlagValueFromProvider(i, "RootDir", "string").(string)
	static := extractFlagValueFromProvider(i, "Static", "[]string").([]string)
	shutdownTimeoutInMinutes := extractFlagValueFromProvider(i, "ShutdownTimeout", "int64").(int64)
	accessLog := extractFlagValueFromProvider(i, "AccessLog", "string").(string)
	outputLog := extractFlagValueFromProvider(i, "OutputLog", "string").(string)
	errorLog := extractFlagValueFromProvider(i, "ErrorLog", "string").(string)
	debug := extractFlagValueFromProvider(i, "Debug", "bool").(bool)
	noBanner := extractFlagValueFromProvider(i, "NoBanner", "bool").(bool)
	cert := extractFlagValueFromProvider(i, "Cert", "string").(string)
	certKey := extractFlagValueFromProvider(i, "CertKey", "string").(string)
	spaPath := extractFlagValueFromProvider(i, "SpaPath", "string").(string)
	noFabricBroker := extractFlagValueFromProvider(i, "NoFabricBroker", "bool").(bool)
	fabricEndpoint := extractFlagValueFromProvider(i, "FabricEndpoint", "string").(string)
	topicPrefix := extractFlagValueFromProvider(i, "TopicPrefix", "string").(string)
	queuePrefix := extractFlagValueFromProvider(i, "QueuePrefix", "string").(string)
	requestPrefix := extractFlagValueFromProvider(i, "RequestPrefix", "string").(string)
	requestQueuePrefix := extractFlagValueFromProvider(i, "RequestQueuePrefix", "string").(string)
	prometheus := extractFlagValueFromProvider(i, "Prometheus", "bool").(bool)

	// if config file flag is provided, read directly from the file
	if len(configFile) > 0 {
		var serverConfig PlatformServerConfig
		b, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &serverConfig); err != nil {
			return nil, err
		}
		return &serverConfig, nil
	}

	// instantiate a server config
	serverConfig := &PlatformServerConfig{
		Host:                     host,
		Port:                     port,
		RootDir:                  rootDir,
		StaticDir:                static,
		ShutdownTimeoutInMinutes: time.Duration(shutdownTimeoutInMinutes),
		LogConfig: &utils.LogConfig{
			AccessLog:     accessLog,
			ErrorLog:      errorLog,
			OutputLog:     outputLog,
			Root:          rootDir,
			FormatOptions: &utils.LogFormatOption{},
		},
		Debug:            debug,
		NoBanner:         noBanner,
		EnablePrometheus: prometheus,
	}

	if len(certKey) > 0 && len(certKey) > 0 {
		var err error
		certKey, err = filepath.Abs(certKey)
		if err != nil {
			return nil, err
		}
		cert, err = filepath.Abs(cert)
		if err != nil {
			return nil, err
		}

		serverConfig.TLSCertConfig = &TLSCertConfig{CertFile: cert, KeyFile: certKey}
	}

	if len(strings.TrimSpace(spaPath)) > 0 {
		var err error
		serverConfig.SpaConfig, err = NewSpaConfig(spaPath)
		if err != nil {
			return nil, err
		}
	}

	// unless --no-fabric-broker flag is provided, set up a broker config
	if !noFabricBroker {
		serverConfig.FabricConfig = &FabricBrokerConfig{
			FabricEndpoint: fabricEndpoint,
			EndpointConfig: &bus.EndpointConfig{
				TopicPrefix:           topicPrefix,
				UserQueuePrefix:       queuePrefix,
				AppRequestPrefix:      requestPrefix,
				AppRequestQueuePrefix: requestQueuePrefix,
				Heartbeat:             60000},
		}
	}

	return serverConfig, nil
}

// marshalResponseBody takes body as an interface not knowing whether it is already converted to []byte or not.
// if it is of a map type then it marshals it using json.Marshal to get the byte representation of it. otherwise
// the input is cast to []byte and returned.
func marshalResponseBody(body interface{}) (bytes []byte, err error) {
	vt := reflect.TypeOf(body)
	if vt == reflect.TypeOf([]byte{}) {
		bytes, err = body.([]byte), nil
	} else {
		bytes, err = json.Marshal(body)
	}

	return
}

// sanitizeConfigRootPath takes *PlatformServerConfig, ensures the path specified by RootDir field exists.
// if RootDir is empty then the current working directory will be populated. if for some reason the path
// cannot be accessed it'll cause a panic.
func sanitizeConfigRootPath(config *PlatformServerConfig) {
	if len(config.RootDir) == 0 {
		wd, _ := os.Getwd()
		config.RootDir = wd
	}

	absRootPath, err := filepath.Abs(config.RootDir)
	if err != nil {
		panic(err)
	}

	_, err = os.Stat(absRootPath)
	if err != nil {
		panic(err)
	}

	// once it has been confirmed that the path exists, set config.RootDir to the absolute path
	config.RootDir = absRootPath
}

func extractFlagValueFromProvider(provider interface{}, key string, parseType string) interface{} {
	switch provider.(type) {
	case *cli.Context:
		cast := provider.(*cli.Context)
		switch parseType {
		case "string":
			return cast.String(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "int":
			return cast.Int(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "int64":
			return cast.Int64(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "[]string":
			return cast.StringSlice(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "bool":
			return cast.Bool(utils.PlatformServerFlagConstants[key]["FlagName"])
		}
		break
	case *serverConfigFactory:
		refl := reflect.ValueOf(provider)
		method := refl.MethodByName(key)
		raw := method.Call([]reflect.Value{})
		return raw[0].Interface()
	}
	return nil
}

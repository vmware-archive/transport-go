package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/pkg/middleware"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"log"
	"net/http"
	"reflect"
	"time"
)

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
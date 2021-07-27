package service

import (
	"fmt"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"net/http"
)

var svcLifecycleManagerInstance ServiceLifecycleManager

type LifecycleHookNotImplementedError struct {
}

func (e *LifecycleHookNotImplementedError) Error() string {
	return fmt.Sprintf("service does not implement lifecycle hook interface")
}

type ServiceLifecycleManager interface {
	GetServiceHooks(serviceChannelName string) (ServiceLifecycleHookEnabled, error)
}

type ServiceLifecycleHookEnabled interface {
	OnServiceReady() chan struct{}            // service initialization logic should be implemented here
	OnServerShutdown()                        // teardown logic goes here and will be automatically invoked on graceful server shutdown
	GetRESTBridgeConfig() []*RESTBridgeConfig // service-to-REST endpoint mappings go here
}

type RESTBridgeConfig struct {
	ServiceChannel       string // transport service channel
	Uri                  string // URI to map the transport service to
	Method               string // HTTP verb to map the transport service request to URI with
	AllowHead            bool   // whether HEAD calls are allowed for this bridge point
	AllowOptions         bool   // whether OPTIONS calls are allowed for this bridge point
	FabricRequestBuilder func(
		w http.ResponseWriter,
		r *http.Request) model.Request // function to transform HTTP request into a transport request
}

type serviceLifecycleManager struct {
	busRef             bus.EventBus // transport bus reference
	serviceRegistryRef ServiceRegistry // service registry reference
}

// GetServiceHooks looks up the ServiceRegistry by service channel and returns the found service
// lifecycle hooks implementation. returns an error if no such service channel exists. also return an
// error if the returned service does not implement the ServiceLifecycleHookEnabled interface.
func (lm *serviceLifecycleManager) GetServiceHooks(serviceChannelName string) (ServiceLifecycleHookEnabled, error) {
	service, err := lm.serviceRegistryRef.GetService(serviceChannelName)
	if err != nil {
		return nil, err
	}

	if lifecycleHookEnabled, ok := service.(ServiceLifecycleHookEnabled); ok {
		return lifecycleHookEnabled, nil
	}
	return nil, &LifecycleHookNotImplementedError{}
}

// GetServiceLifecycleManager returns a singleton instance of ServiceLifecycleManager
func GetServiceLifecycleManager(bus bus.EventBus, serviceRegistry ServiceRegistry) ServiceLifecycleManager {
	if svcLifecycleManagerInstance == nil {
		svcLifecycleManagerInstance = &serviceLifecycleManager{
			busRef:             bus,
			serviceRegistryRef: serviceRegistry,
		}
	}
	return svcLifecycleManagerInstance
}

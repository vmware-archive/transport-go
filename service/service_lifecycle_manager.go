package service

import (
	"fmt"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"net/http"
	"sync"
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
	OnServiceReady() chan struct{} // service initialization logic should be implemented here
	OnServerShutdown() // teardown logic goes here and will be automatically invoked on graceful server shutdown
	GetRESTBridgeConfig() []*RESTBridgeConfig // service-to-REST endpoint mappings go here
}

type RESTBridgeConfig struct {
	ServiceChannel string
	Uri string
	Method string
	FabricRequestBuilder func(w http.ResponseWriter, r *http.Request) model.Request
}

type serviceLifecycleManager struct {
	busRef bus.EventBus
	serviceRegistryRef ServiceRegistry
	mu       sync.Mutex
}

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
			busRef:   bus,
			serviceRegistryRef: serviceRegistry,
			mu:       sync.Mutex{},
		}
	}
	return svcLifecycleManagerInstance
}
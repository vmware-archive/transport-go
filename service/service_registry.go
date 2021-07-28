// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"fmt"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"log"
	"reflect"
	"sync"
)

var internalServices = map[string]bool{
	"fabric-rest": true,
}

const (
	LifecycleManagerChannelName = "service-lifecycle-manager"

	// store constants
	ServiceReadyStore = "service-ready-notification-store"
	ServiceInitStateChange = "service-init-state-change"
)

// ServiceRegistry is the registry for  all local fabric services.
type ServiceRegistry interface {
	// GetAllServiceChannels returns all active Fabric service channels as a slice of strings
	GetAllServiceChannels() []string

	// RegisterService registers a new fabric service and associates it with a given EventBus channel.
	// Only one fabric service can be associated with a given channel.
	// If the fabric service implements the FabricInitializableService interface
	// its Init method will be called during the registration process.
	RegisterService(service FabricService, serviceChannelName string) error

	// UnregisterService unregisters the fabric service associated with the given channel.
	UnregisterService(serviceChannelName string) error

	// SetGlobalRestServiceBaseHost sets the global base host or host:port to be used by the restService
	SetGlobalRestServiceBaseHost(host string)

	// GetService returns the FabricService for the channel name given as the parameter
	GetService(serviceChannelName string) (FabricService, error)
}

type serviceRegistry struct {
	lock     sync.Mutex
	services map[string]*fabricServiceWrapper
	bus      bus.EventBus
}

var once sync.Once
var registry ServiceRegistry

func GetServiceRegistry() ServiceRegistry {
	once.Do(func() {
		registry = newServiceRegistry(bus.GetBus())

		// ensure a new service lifecycle manager is initialized as early as possible
		svcLifecycleManagerInstance = nil
		GetServiceLifecycleManager()
	})
	return registry
}

func newServiceRegistry(bus bus.EventBus) ServiceRegistry {
	registry := &serviceRegistry{
		bus:      bus,
		services: make(map[string]*fabricServiceWrapper),
	}
	// create a channel for service lifecycle manager
	_ = bus.GetChannelManager().CreateChannel(LifecycleManagerChannelName)

	// create a bus store for delivering service ready notifications
	bus.GetStoreManager().CreateStoreWithType(ServiceReadyStore, reflect.TypeOf(true)).Initialize()

	// auto-register the restService
	registry.RegisterService(&restService{}, restServiceChannel)
	return registry
}

// GetService returns the FabricService instance registered at the provided service channel name.
// if no service is found at the service channel it returns an error.
func (r *serviceRegistry) GetService(serviceChannelName string) (FabricService, error) {
	if serviceWrapper, ok := r.services[serviceChannelName]; ok {
		return serviceWrapper.service, nil
	}
	return nil, fmt.Errorf("fabric service not found at channel %s", serviceChannelName)
}

func (r *serviceRegistry) SetGlobalRestServiceBaseHost(host string) {
	r.services[restServiceChannel].service.(*restService).setBaseHost(host)
}

// GetAllServiceChannels returns the list of service channels that are registered with the registry
func (r *serviceRegistry) GetAllServiceChannels() []string {
	services := make([]string, 0)
	for chanName, _ := range r.services {
		// do not return internal services like fabric-rest
		if isInternal, found := internalServices[chanName]; !isInternal || !found {
			services = append(services, chanName)
		}
	}
	return services
}

func (r *serviceRegistry) RegisterService(service FabricService, serviceChannelName string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if service == nil {
		return fmt.Errorf("unable to register service: nil service")
	}

	if _, ok := r.services[serviceChannelName]; ok {
		return fmt.Errorf("unable to register service: service channel name is already used: %s", serviceChannelName)
	}

	sw := newServiceWrapper(r.bus, service, serviceChannelName)
	err := sw.init()
	if err != nil {
		return err
	}

	r.services[serviceChannelName] = sw

	// if the service is an internal service like fabric-rest don't bother setting up lifecycle hooks
	if isInternal, _ := internalServices[serviceChannelName]; isInternal {
		return nil
	}

	// see if the service implements ServiceLifecycleHookEnabled interface and set up REST bridges as configured
	var hooks ServiceLifecycleHookEnabled
	lcm := GetServiceLifecycleManager()

	// hand off registering REST bridges to Plank via bus messages
	if hooks = lcm.GetServiceHooks(serviceChannelName); hooks != nil {
		if err = bus.GetBus().SendResponseMessage(
			LifecycleManagerChannelName,
			&SetupRESTBridgeRequest{ServiceChannel: serviceChannelName, Config: hooks.GetRESTBridgeConfig()},
			bus.GetBus().GetId()); err != nil {
			return err
		}
	}

	return nil
}

func (r *serviceRegistry) UnregisterService(serviceChannelName string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	sw, ok := r.services[serviceChannelName]
	if !ok {
		return fmt.Errorf("unable to unregister service: no service is registered for channel \"%s\"", serviceChannelName)
	}
	sw.unregister()
	delete(r.services, serviceChannelName)
	return nil
}

type fabricServiceWrapper struct {
	service           FabricService
	fabricCore        *fabricCore
	requestMsgHandler bus.MessageHandler
}

func newServiceWrapper(
	bus bus.EventBus, service FabricService, serviceChannelName string) *fabricServiceWrapper {

	return &fabricServiceWrapper{
		service: service,
		fabricCore: &fabricCore{
			bus:         bus,
			channelName: serviceChannelName,
		},
	}
}

func (sw *fabricServiceWrapper) init() error {
	sw.fabricCore.bus.GetChannelManager().CreateChannel(sw.fabricCore.channelName)

	initializationService, ok := sw.service.(FabricInitializableService)
	if ok {
		initializationErr := initializationService.Init(sw.fabricCore)
		if initializationErr != nil {
			return initializationErr
		}
	}

	mh, err := sw.fabricCore.bus.ListenRequestStream(sw.fabricCore.channelName)
	if err != nil {
		return err
	}

	sw.requestMsgHandler = mh
	mh.Handle(
		func(message *model.Message) {
			requestPtr, ok := message.Payload.(*model.Request)
			if !ok {
				request, ok := message.Payload.(model.Request)
				if !ok {
					log.Println("cannot cast service request payload to model.Request")
					return
				}
				requestPtr = &request
			}

			if message.DestinationId != nil {
				requestPtr.Id = message.DestinationId
			}

			sw.service.HandleServiceRequest(requestPtr, sw.fabricCore)
		},
		func(e error) {})

	return nil
}

func (sw *fabricServiceWrapper) unregister() {
	if sw.requestMsgHandler != nil {
		sw.requestMsgHandler.Close()
	}
}

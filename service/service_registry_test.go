// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
    "errors"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/vmware/transport-go/bus"
    "github.com/vmware/transport-go/model"
    "net/http"
    "sync"
    "testing"
)

func newTestServiceRegistry() *serviceRegistry {
    eventBus := bus.NewEventBusInstance()
    return newServiceRegistry(eventBus).(*serviceRegistry)
}

func newTestServiceLifecycleManager(sr ServiceRegistry) ServiceLifecycleManager {
    return newServiceLifecycleManager(sr)
}

type mockFabricService struct {
    processedRequests []*model.Request
    core FabricServiceCore
    wg sync.WaitGroup
}

func (fs *mockFabricService) HandleServiceRequest(request *model.Request, core FabricServiceCore) {
    fs.processedRequests = append(fs.processedRequests, request)
    fs.core = core
    fs.wg.Done()
}

type mockLifecycleHookEnabledService struct {
    initChan chan bool
    core FabricServiceCore
    shutdown bool
}

func (s *mockLifecycleHookEnabledService) HandleServiceRequest(request *model.Request, core FabricServiceCore) {
}

func (s *mockLifecycleHookEnabledService) OnServiceReady() chan bool {
    s.initChan = make(chan bool, 1)
    s.initChan <- true
    return s.initChan
}

func (s *mockLifecycleHookEnabledService) OnServerShutdown() {
    s.shutdown = true
}

func (s *mockLifecycleHookEnabledService) GetRESTBridgeConfig() []*RESTBridgeConfig {
    return []*RESTBridgeConfig{
        {
            ServiceChannel:       "another-test-channel",
            Uri:                  "/rest/test",
            Method:               http.MethodGet,
            AllowHead:            true,
            AllowOptions:         true,
            FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
                return model.Request{
                    Id: &uuid.UUID{},
                    Payload: "test",
                }
            },
        },
    }
}

type mockInitializableService struct {
    initialized bool
    core        FabricServiceCore
    initError   error
}

func (fs *mockInitializableService) Init(core FabricServiceCore) error {
    fs.core = core
    fs.initialized = true
    return fs.initError
}

func (fs *mockInitializableService) HandleServiceRequest(request *model.Request, core FabricServiceCore) {
}

func TestGetServiceRegistry(t *testing.T) {
    sr := GetServiceRegistry()
    sr2 := GetServiceRegistry()
    assert.NotNil(t, sr)
    assert.Equal(t, sr, sr2)
}

func TestServiceRegistry_RegisterService(t *testing.T) {
    registry := newTestServiceRegistry()
    mockService := &mockFabricService{}

    assert.Nil(t, registry.RegisterService(mockService, "test-channel"))
    assert.True(t, registry.bus.GetChannelManager().CheckChannelExists("test-channel"))

    id := uuid.New()
    req := model.Request{
        Id: &id,
        Request: "test-request",
        Payload: "request-payload",
    }

    mockService.wg.Add(1)
    registry.bus.SendRequestMessage("test-channel", req, nil)
    mockService.wg.Wait()

    assert.Equal(t, len(mockService.processedRequests), 1)
    assert.Equal(t, *mockService.processedRequests[0], req)
    assert.NotNil(t, mockService.core)

    registry.bus.SendRequestMessage("test-channel", "invalid-request", nil)
    registry.bus.SendRequestMessage("test-channel", nil, nil)
    registry.bus.SendResponseMessage("test-channel", req, nil)
    registry.bus.SendErrorMessage("test-channel", errors.New("test-error"), nil)

    mockService.wg.Add(1)
    registry.bus.SendRequestMessage("test-channel", &req, nil)
    mockService.wg.Wait()

    assert.Equal(t, len(mockService.processedRequests), 2)
    assert.Equal(t, mockService.processedRequests[1], &req)
    assert.NotNil(t, mockService.core)

    mockService.wg.Add(1)
    uuid := uuid.New()
    registry.bus.SendRequestMessage("test-channel", model.Request{
        Request: "test-request-2",
        Payload: "request-payload",
    }, &uuid)
    mockService.wg.Wait()

    assert.Equal(t, len(mockService.processedRequests), 3)
    assert.Equal(t, mockService.processedRequests[2].Id, &uuid)

    assert.EqualError(t, registry.RegisterService(&mockFabricService{}, "test-channel"),
        "unable to register service: service channel name is already used: test-channel")

    assert.EqualError(t, registry.RegisterService(nil, "test-channel2"),
        "unable to register service: nil service")

    assert.False(t, registry.bus.GetChannelManager().CheckChannelExists("test-channel2"))
}

func TestServiceRegistry_RegisterInitializableService(t *testing.T) {
    registry := newTestServiceRegistry()
    mockService := &mockInitializableService{}
    assert.Nil(t, registry.RegisterService(mockService, "test-channel"))

    assert.True(t, mockService.initialized)
    assert.NotNil(t, mockService.core)

    assert.EqualError(t,
        registry.RegisterService(&mockInitializableService{initError: errors.New("init-error")}, "test-channel2"),
        "init-error")
}

func TestServiceRegistry_UnregisterService(t *testing.T) {
    registry := newTestServiceRegistry()
    mockService := &mockFabricService{}

    assert.Nil(t, registry.RegisterService(mockService, "test-channel"))
    assert.True(t, registry.bus.GetChannelManager().CheckChannelExists("test-channel"))

    id := uuid.New()
    req := model.Request{
        Id: &id,
        Request: "test-request",
        Payload: "request-payload",
    }

    assert.Nil(t, registry.UnregisterService("test-channel"))
    registry.bus.SendRequestMessage("test-channel", req, nil)

    assert.Equal(t, len(mockService.processedRequests), 0)
    assert.EqualError(t, registry.UnregisterService("test-channel"),
        "unable to unregister service: no service is registered for channel \"test-channel\"")
}

func TestServiceRegistry_SetGlobalRestServiceBaseHost(t *testing.T) {
    registry := newTestServiceRegistry()
    registry.SetGlobalRestServiceBaseHost("localhost:9999")
    assert.Equal(t, "localhost:9999",
            registry.services[restServiceChannel].service.(*restService).baseHost)
}

func TestServiceRegistry_GetAllServiceChannels(t *testing.T) {
    registry := newTestServiceRegistry()
    mockService := &mockFabricService{}

    registry.RegisterService(mockService, "test-channel")
    chans := registry.GetAllServiceChannels()

    assert.Len(t, chans, 1)
    assert.EqualValues(t, "test-channel", chans[0])
}

func TestServiceRegistry_RegisterService_LifecycleHookEnabled(t *testing.T) {
    svc := &mockLifecycleHookEnabledService{}
    registry := newTestServiceRegistry()
    registry.RegisterService(svc, "another-test-channel")

    assert.True(t, <-svc.OnServiceReady())

    svc.OnServerShutdown()
    assert.True(t, svc.shutdown)

    restBridgeConfig := svc.GetRESTBridgeConfig()
    assert.NotNil(t, restBridgeConfig)
}
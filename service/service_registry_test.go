// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "go-bifrost/model"
    "go-bifrost/bus"
    "sync"
    "github.com/google/uuid"
    "errors"
)

func newTestServiceRegistry() *serviceRegistry {
    eventBus := bus.NewEventBusInstance()
    return NewServiceRegistry(eventBus).(*serviceRegistry)
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
    assert.Equal(t, mockService.processedRequests[0], &req)
    assert.NotNil(t, mockService.core)

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
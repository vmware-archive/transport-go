package service

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/transport-go/model"
	"net/http"
	"sync"
	"testing"
)

func TestServiceLifecycleManager(t *testing.T) {
	// arrange
	sr := newTestServiceRegistry()

	// act
	lcm := newTestServiceLifecycleManager(sr)

	// assert
	assert.NotNil(t, lcm)
}

func TestServiceLifecycleManager_GetServiceHooks(t *testing.T) {
	// arrange
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	svc := &mockLifecycleHookEnabledService{}
	sr.RegisterService(svc, "another-test-channel")

	// act
	hooks := lcm.GetServiceHooks("another-test-channel")

	// assert
	assert.NotNil(t, hooks)
}

func TestServiceLifecycleManager_GetServiceHooks_NoSuchService(t *testing.T) {
	// arrange
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)

	// act
	hooks := lcm.GetServiceHooks("i-don-t-exist")

	// assert
	assert.Nil(t, hooks)
}

func TestServiceLifecycleManager_GetServiceHooks_LifecycleHooksNotImplemented(t *testing.T) {
	// arrange
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	svc := &mockInitializableService{}
	sr.RegisterService(svc, "test-channel")

	// act
	hooks := lcm.GetServiceHooks("test-channel")

	// assert
	assert.Nil(t, hooks)
}

func TestServiceLifecycleManager_OverrideRESTBridgeConfig_NoSuchService(t *testing.T) {
	// arrange
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)

	// act
	err := lcm.OverrideRESTBridgeConfig("no-such-service", []*RESTBridgeConfig{})

	// assert
	assert.NotNil(t, err)
}

func TestServiceLifecycleManager_OverrideRESTBridgeConfig(t *testing.T) {
	// arrange
	wg := sync.WaitGroup{}
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	sr.lifecycleManager = lcm.(*serviceLifecycleManager)
	_ = sr.RegisterService(&mockLifecycleHookEnabledService{}, "another-test-channel")

	// arrange: test payload
	payload := &RESTBridgeConfig{
		ServiceChannel:       "another-test-channel",
		Uri:                  "/rest/new-uri",
		Method:               http.MethodGet,
		AllowHead:            false,
		AllowOptions:         false,
		FabricRequestBuilder: nil,
	}

	// arrange: set up a handler to expect to receive a payload that matches the test payload set above
	stream, err := sr.bus.ListenStreamForDestination(LifecycleManagerChannelName, sr.bus.GetId())
	assert.Nil(t, err)
	defer stream.Close()

	wg.Add(1)
	stream.Handle(func(message *model.Message) {
		req, parsed := message.Payload.(*SetupRESTBridgeRequest)
		if !parsed {
			assert.Fail(t, "should have expected *SetupRESTBridgeRequest payload")
		}
		// assert
		assert.True(t, req.Override)
		assert.EqualValues(t, "another-test-channel", req.ServiceChannel)
		assert.EqualValues(t, payload, req.Config[0])
		wg.Done()
	}, func(err error) {
		assert.Fail(t, "should not have errored", err)
	})

	// act
	err = lcm.OverrideRESTBridgeConfig("another-test-channel", []*RESTBridgeConfig{payload})
	assert.Nil(t, err)
	wg.Wait()
}

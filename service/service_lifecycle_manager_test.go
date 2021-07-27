package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServiceLifecycleManager(t *testing.T) {
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	assert.NotNil(t, lcm)
}

func TestServiceLifecycleManager_GetServiceHooks(t *testing.T) {
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	svc := &mockLifecycleHookEnabledService{}
	sr.RegisterService(svc, "another-test-channel")
	hooks := lcm.GetServiceHooks("another-test-channel")
	assert.NotNil(t, hooks)
}

func TestServiceLifecycleManager_GetServiceHooks_NoSuchService(t *testing.T) {
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	hooks := lcm.GetServiceHooks("i-don-t-exist")
	assert.Nil(t, hooks)
}

func TestServiceLifecycleManager_GetServiceHooks_LifecycleHooksNotImplemented(t *testing.T) {
	sr := newTestServiceRegistry()
	lcm := newTestServiceLifecycleManager(sr)
	svc := &mockInitializableService{}
	sr.RegisterService(svc, "test-channel")
	hooks := lcm.GetServiceHooks("test-channel")
	assert.Nil(t, hooks)
}

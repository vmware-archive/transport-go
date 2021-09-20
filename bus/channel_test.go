// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/model"
	"testing"
)

var testChannelName string = "testing"

func TestChannel_CheckChannelCreation(t *testing.T) {

	channel := NewChannel(testChannelName)
	assert.Empty(t, channel.eventHandlers)

}

func TestChannel_SubscribeHandler(t *testing.T) {
	id := uuid.New()
	channel := NewChannel(testChannelName)
	handler := func(*model.Message) {}
	channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

	assert.Equal(t, 1, len(channel.eventHandlers))

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

	assert.Equal(t, 2, len(channel.eventHandlers))
}

func TestChannel_HandlerCheck(t *testing.T) {
	channel := NewChannel(testChannelName)
	id := uuid.New()
	assert.False(t, channel.ContainsHandlers())

	handler := func(*model.Message) {}
	channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

	assert.True(t, channel.ContainsHandlers())
}

func TestChannel_SendMessage(t *testing.T) {
	id := uuid.New()
	channel := NewChannel(testChannelName)
	handler := func(message *model.Message) {
		assert.Equal(t, message.Payload.(string), "pickled eggs")
		assert.Equal(t, message.Channel, testChannelName)
	}

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

	var message = &model.Message{
		Id:        &id,
		Payload:   "pickled eggs",
		Channel:   testChannelName,
		Direction: model.RequestDir}

	channel.Send(message)
	channel.wg.Wait()
}

func TestChannel_SendMessageRunOnceHasRun(t *testing.T) {
	id := uuid.New()
	channel := NewChannel(testChannelName)
	count := 0
	handler := func(message *model.Message) {
		assert.Equal(t, message.Payload.(string), "pickled eggs")
		assert.Equal(t, message.Channel, testChannelName)
		count++
	}

	h := &channelEventHandler{callBackFunction: handler, runOnce: true, uuid: &id}
	channel.subscribeHandler(h)

	var message = &model.Message{
		Id:        &id,
		Payload:   "pickled eggs",
		Channel:   testChannelName,
		Direction: model.RequestDir}

	channel.Send(message)
	channel.wg.Wait()
	h.runCount = 1
	channel.Send(message)
	assert.Len(t, channel.eventHandlers, 0)
	assert.Equal(t, 1, count)
}

func TestChannel_SendMultipleMessages(t *testing.T) {
	id := uuid.New()
	channel := NewChannel(testChannelName)
	var counter int32 = 0
	handler := func(message *model.Message) {
		assert.Equal(t, message.Payload.(string), "chewy louie")
		assert.Equal(t, message.Channel, testChannelName)
		inc(&counter)
	}

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})
	var message = &model.Message{
		Id:        &id,
		Payload:   "chewy louie",
		Channel:   testChannelName,
		Direction: model.RequestDir}

	channel.Send(message)
	channel.Send(message)
	channel.Send(message)
	channel.wg.Wait()
	assert.Equal(t, int32(3), counter)
}

func TestChannel_MultiHandlerSingleMessage(t *testing.T) {
	id := uuid.New()
	channel := NewChannel(testChannelName)
	var counterA, counterB, counterC int32 = 0, 0, 0

	handlerA := func(message *model.Message) {
		inc(&counterA)
	}
	handlerB := func(message *model.Message) {
		inc(&counterB)
	}
	handlerC := func(message *model.Message) {
		inc(&counterC)
	}

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handlerA, runOnce: false, uuid: &id})

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handlerB, runOnce: false, uuid: &id})

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handlerC, runOnce: false, uuid: &id})

	var message = &model.Message{
		Id:        &id,
		Payload:   "late night munchies",
		Channel:   testChannelName,
		Direction: model.RequestDir}

	channel.Send(message)
	channel.Send(message)
	channel.Send(message)
	channel.wg.Wait()
	value := counterA + counterB + counterC

	assert.Equal(t, int32(9), value)
}

func TestChannel_Privacy(t *testing.T) {
	channel := NewChannel(testChannelName)
	assert.False(t, channel.private)
	channel.SetPrivate(true)
	assert.True(t, channel.IsPrivate())
}

func TestChannel_ChannelGalactic(t *testing.T) {
	channel := NewChannel(testChannelName)
	assert.False(t, channel.galactic)
	channel.SetGalactic("somewhere")
	assert.True(t, channel.IsGalactic())
}

func TestChannel_RemoveEventHandler(t *testing.T) {
	channel := NewChannel(testChannelName)
	handlerA := func(message *model.Message) {}
	handlerB := func(message *model.Message) {}

	idA := uuid.New()
	idB := uuid.New()

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handlerA, runOnce: false, uuid: &idA})

	channel.subscribeHandler(&channelEventHandler{callBackFunction: handlerB, runOnce: false, uuid: &idB})

	assert.Len(t, channel.eventHandlers, 2)

	// remove the first handler (A)
	channel.removeEventHandler(0)
	assert.Len(t, channel.eventHandlers, 1)
	assert.Equal(t, idB.String(), channel.eventHandlers[0].uuid.String())

	// remove the second handler B)
	channel.removeEventHandler(0)
	assert.True(t, len(channel.eventHandlers) == 0)

}

func TestChannel_RemoveEventHandlerNoHandlers(t *testing.T) {
	channel := NewChannel(testChannelName)

	channel.removeEventHandler(0)
	assert.Len(t, channel.eventHandlers, 0)
}

func TestChannel_RemoveEventHandlerOOBIndex(t *testing.T) {
	channel := NewChannel(testChannelName)
	id := uuid.New()
	handler := func(*model.Message) {}
	channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

	channel.removeEventHandler(999)
	assert.Len(t, channel.eventHandlers, 1)
}

func TestChannel_AddRemoveBrokerSubscription(t *testing.T) {
	channel := NewChannel(testChannelName)
	id := uuid.New()
	sub := &MockBridgeSubscription{Id: &id}
	c := &MockBridgeConnection{Id: &id}
	channel.addBrokerSubscription(c, sub)
	assert.Len(t, channel.brokerSubs, 1)
	channel.removeBrokerSubscription(sub)
	assert.Len(t, channel.brokerSubs, 0)
}

func TestChannel_CheckIfBrokerSubscribed(t *testing.T) {

	cId := uuid.New()
	sId := uuid.New()
	sId2 := uuid.New()

	c := &MockBridgeConnection{
		Id: &cId,
	}
	s := &MockBridgeSubscription{Id: &sId}
	s2 := &MockBridgeSubscription{Id: &sId2}

	cm := NewBusChannelManager(GetBus())
	ch := cm.CreateChannel("testing-broker-subs")
	ch.addBrokerSubscription(c, s)
	assert.True(t, ch.isBrokerSubscribed(s))
	assert.False(t, ch.isBrokerSubscribed(s2))

	ch.removeBrokerSubscription(s)
	assert.False(t, ch.isBrokerSubscribed(s))
}

type MockBridgeConnection struct {
	mock.Mock
	Id *uuid.UUID
}

func (c *MockBridgeConnection) GetId() *uuid.UUID {
	return c.Id
}

func (c *MockBridgeConnection) SubscribeReplyDestination(destination string) (bridge.Subscription, error) {
	args := c.MethodCalled("Subscribe", destination)
	return args.Get(0).(bridge.Subscription), args.Error(1)
}

func (c *MockBridgeConnection) Subscribe(destination string) (bridge.Subscription, error) {
	args := c.MethodCalled("Subscribe", destination)
	return args.Get(0).(bridge.Subscription), args.Error(1)
}

func (c *MockBridgeConnection) Disconnect() (err error) {
	return nil
}

func (c *MockBridgeConnection) SendJSONMessage(destination string, payload []byte, opts ...func(frame *frame.Frame) error) error {
	args := c.MethodCalled("SendJSONMessage", destination, payload)
	return args.Error(0)
}

func (c *MockBridgeConnection) SendMessage(destination, contentType string, payload []byte, opts ...func(frame *frame.Frame) error) error {
	args := c.MethodCalled("SendMessage", destination, contentType, payload)
	return args.Error(0)
}

func (c *MockBridgeConnection) SendMessageWithReplyDestination(destination, reply, contentType string, payload []byte, opts ...func(frame *frame.Frame) error) error {
	args := c.MethodCalled("SendMessage", destination, contentType, payload)
	return args.Error(0)
}

type MockBridgeSubscription struct {
	Id          *uuid.UUID
	Destination string
	Channel     chan *model.Message
}

func (m *MockBridgeSubscription) GetId() *uuid.UUID {
	return m.Id
}

func (m *MockBridgeSubscription) GetDestination() string {
	return m.Destination
}

func (m *MockBridgeSubscription) GetMsgChannel() chan *model.Message {
	return m.Channel
}

func (m *MockBridgeSubscription) Unsubscribe() error {
	return nil
}

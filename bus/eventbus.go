// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/stompserver"
	"sync"
	"sync/atomic"
)

const TRANSPORT_INTERNAL_CHANNEL_PREFIX = "_transportInternal/"

// EventBus provides access to ChannelManager, simple message sending and simple API calls for handling
// messaging and error handling over channels on the bus.
type EventBus interface {
	GetId() *uuid.UUID
	GetChannelManager() ChannelManager
	SendRequestMessage(channelName string, payload interface{}, destinationId *uuid.UUID) error
	SendResponseMessage(channelName string, payload interface{}, destinationId *uuid.UUID) error
	SendBroadcastMessage(channelName string, payload interface{}) error
	SendErrorMessage(channelName string, err error, destinationId *uuid.UUID) error
	ListenStream(channelName string) (MessageHandler, error)
	ListenStreamForDestination(channelName string, destinationId *uuid.UUID) (MessageHandler, error)
	ListenFirehose(channelName string) (MessageHandler, error)
	ListenRequestStream(channelName string) (MessageHandler, error)
	ListenRequestStreamForDestination(channelName string, destinationId *uuid.UUID) (MessageHandler, error)
	ListenRequestOnce(channelName string) (MessageHandler, error)
	ListenRequestOnceForDestination(channelName string, destinationId *uuid.UUID) (MessageHandler, error)
	ListenOnce(channelName string) (MessageHandler, error)
	ListenOnceForDestination(channelName string, destId *uuid.UUID) (MessageHandler, error)
	RequestOnce(channelName string, payload interface{}) (MessageHandler, error)
	RequestOnceForDestination(channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error)
	RequestStream(channelName string, payload interface{}) (MessageHandler, error)
	RequestStreamForDestination(channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error)
	ConnectBroker(config *bridge.BrokerConnectorConfig) (conn bridge.Connection, err error)
	StartFabricEndpoint(connectionListener stompserver.RawConnectionListener, config EndpointConfig) error
	StopFabricEndpoint() error
	GetStoreManager() StoreManager
	CreateSyncTransaction() BusTransaction
	CreateAsyncTransaction() BusTransaction
	AddMonitorEventListener(listener MonitorEventHandler, eventTypes ...MonitorEventType) MonitorEventListenerId
	RemoveMonitorEventListener(listenerId MonitorEventListenerId)
	SendMonitorEvent(evtType MonitorEventType, entityName string, data interface{})
}

var enableLogging bool = false

func EnableLogging(enable bool) {
	enableLogging = enable
}

var once sync.Once
var busInstance EventBus

// ResetBus destroys existing bus instance and creates a new one
func ResetBus() EventBus {
	once = sync.Once{}
	return GetBus()
}

// Get a reference to the EventBus.
func GetBus() EventBus {
	once.Do(func() {
		busInstance = NewEventBusInstance()
	})
	return busInstance
}

func NewEventBusInstance() EventBus {
	bf := new(transportEventBus)
	bf.init()
	return bf
}

type transportEventBus struct {
	ChannelManager    ChannelManager
	storeManager      StoreManager
	Id                uuid.UUID
	brokerConnections map[*uuid.UUID]bridge.Connection
	bc                bridge.BrokerConnector
	fabEndpoint       FabricEndpoint
	initStoreSync     sync.Once
	storeSyncService  *storeSyncService
	monitor           *transportMonitor
}

type MonitorEventListenerId int

type transportMonitor struct {
	lock                  sync.RWMutex
	listenersByType       map[MonitorEventType]map[MonitorEventListenerId]MonitorEventHandler
	listenersForAllEvents map[MonitorEventListenerId]MonitorEventHandler
	subId                 MonitorEventListenerId
}

func newMonitor() *transportMonitor {
	return &transportMonitor{
		listenersByType:       make(map[MonitorEventType]map[MonitorEventListenerId]MonitorEventHandler),
		listenersForAllEvents: make(map[MonitorEventListenerId]MonitorEventHandler),
	}
}

func (m *transportMonitor) addListener(listener MonitorEventHandler, eventTypes []MonitorEventType) MonitorEventListenerId {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.subId++
	if len(eventTypes) == 0 {
		m.listenersForAllEvents[m.subId] = listener
	} else {
		for _, eventType := range eventTypes {
			listeners, ok := m.listenersByType[eventType]
			if !ok {
				listeners = make(map[MonitorEventListenerId]MonitorEventHandler)
				m.listenersByType[eventType] = listeners
			}
			listeners[m.subId] = listener
		}
	}

	return m.subId
}

func (m *transportMonitor) removeListener(listenerId MonitorEventListenerId) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.listenersForAllEvents, listenerId)
	for _, listeners := range m.listenersByType {
		delete(listeners, listenerId)
	}
}

func (m *transportMonitor) sendEvent(event *MonitorEvent) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, l := range m.listenersForAllEvents {
		l(event)
	}

	for _, l := range m.listenersByType[event.EventType] {
		l(event)
	}
}

func (bus *transportEventBus) GetId() *uuid.UUID {
	return &bus.Id
}

func (bus *transportEventBus) init() {
	bus.Id = uuid.New()
	bus.storeManager = newStoreManager(bus)
	bus.ChannelManager = NewBusChannelManager(bus)
	bus.brokerConnections = make(map[*uuid.UUID]bridge.Connection)
	bus.bc = bridge.NewBrokerConnector()
	bus.monitor = newMonitor()
	if enableLogging {
		fmt.Printf("🌈 Transport booted with Id [%s]\n", bus.Id.String())
	}
}

func (bus *transportEventBus) GetStoreManager() StoreManager {
	return bus.storeManager
}

// GetChannelManager Get a pointer to the ChannelManager for managing Channels.
func (bus *transportEventBus) GetChannelManager() ChannelManager {
	return bus.ChannelManager
}

// SendResponseMessage Send a ResponseDir type (inbound) message on Channel, with supplied Payload.
// Throws error if the Channel does not exist.
func (bus *transportEventBus) SendResponseMessage(channelName string, payload interface{}, destId *uuid.UUID) error {
	channelObject, err := bus.ChannelManager.GetChannel(channelName)
	if err != nil {
		return err
	}
	config := buildConfig(channelName, payload, destId)
	message := model.GenerateResponse(config)
	sendMessageToChannel(channelObject, message)
	return nil
}

// SendBroadcastMessage sends the payload as an outbound broadcast message to channelName. Since it is a broadcast,
// the payload does not require a destination ID. Throws an error if the channel does not exist.
func (bus *transportEventBus) SendBroadcastMessage(channelName string, payload interface{}) error {
	channelObject, err := bus.ChannelManager.GetChannel(channelName)
	if err != nil {
		return err
	}
	config := buildConfig(channelName, payload, nil)
	message := model.GenerateResponse(config)
	sendMessageToChannel(channelObject, message)
	return nil
}

// SendRequestMessage Send a RequestDir type message (outbound) message on Channel, with supplied Payload.
// Throws error if the Channel does not exist.
func (bus *transportEventBus) SendRequestMessage(channelName string, payload interface{}, destId *uuid.UUID) error {
	channelObject, err := bus.ChannelManager.GetChannel(channelName)
	if err != nil {
		return err
	}
	config := buildConfig(channelName, payload, destId)
	message := model.GenerateRequest(config)
	sendMessageToChannel(channelObject, message)
	return nil
}

// SendErrorMessage Send a ErrorDir type message (outbound) message on Channel, with supplied error
// Throws error if the Channel does not exist.
func (bus *transportEventBus) SendErrorMessage(channelName string, err error, destId *uuid.UUID) error {
	channelObject, chanErr := bus.ChannelManager.GetChannel(channelName)
	if chanErr != nil {
		return err
	}
	config := buildError(channelName, err, destId)
	message := model.GenerateError(config)
	sendMessageToChannel(channelObject, message)
	return nil
}

// ListenStream Listen to stream of ResponseDir (inbound) messages on Channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *transportEventBus) ListenStream(channelName string) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, true, false, nil, false)
	return messageHandler, nil
}

// AddMonitorEventListener Adds new monitor event listener for the a given set of event types.
// If eventTypes param is not provided, the listener will be called for all events.
// Returns the id of the newly added event listener.
func (bus *transportEventBus) AddMonitorEventListener(
	listener MonitorEventHandler, eventTypes ...MonitorEventType) MonitorEventListenerId {

	return bus.monitor.addListener(listener, eventTypes)
}

// RemoveMonitorEventListener Removes a given event listener
func (bus *transportEventBus) RemoveMonitorEventListener(listenerId MonitorEventListenerId) {
	bus.monitor.removeListener(listenerId)
}

// ListenStreamForDestination Listen to stream of ResponseDir (inbound) messages on Channel for a specific DestinationId.
// Will keep on ticking until closed, returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *transportEventBus) ListenStreamForDestination(channelName string, destId *uuid.UUID) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	if destId == nil {
		return nil, fmt.Errorf("DestinationId cannot be nil")
	}
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, false, false, destId, false)
	return messageHandler, nil
}

// ListenRequestStream Listen to a stream of RequestDir (outbound) messages on Channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenRequestStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *transportEventBus) ListenRequestStream(channelName string) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, true, false, nil, false)
	return messageHandler, nil
}

// ListenRequestStreamForDestination Listen to a stream of RequestDir (outbound) messages on Channel for a specific DestinationId.
// Will keep on ticking until closed, returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenRequestStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *transportEventBus) ListenRequestStreamForDestination(
	channelName string, destId *uuid.UUID) (MessageHandler, error) {

	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	if destId == nil {
		return nil, fmt.Errorf("DestinationId cannot be nil")
	}
	messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, false, false, destId, false)
	return messageHandler, nil
}

// ListenRequestOnce Listen for a single RequestDir (outbound) messages on Channel. Handler is closed after a single event.
// Returns MessageHandler
func (bus *transportEventBus) ListenRequestOnce(channelName string) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	id := checkForSuppliedId(nil)
	messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, true, false, id, true)
	return messageHandler, nil
}

// ListenRequestOnceForDestination Listen for a single RequestDir (outbound) messages on Channel with a specific DestinationId.
// Handler is closed after a single event, returns MessageHandler
func (bus *transportEventBus) ListenRequestOnceForDestination(
	channelName string, destId *uuid.UUID) (MessageHandler, error) {

	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	if destId == nil {
		return nil, fmt.Errorf("DestinationId cannot be nil")
	}
	messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, false, false, destId, true)
	return messageHandler, nil
}

// ListenFirehose pull in everything being fired on a channel.
func (bus *transportEventBus) ListenFirehose(channelName string) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, true, true, nil, false)
	return messageHandler, nil
}

// ListenOnce Will listen for a single ResponseDir message on the Channel before un-subscribing automatically.
func (bus *transportEventBus) ListenOnce(channelName string) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	id := checkForSuppliedId(nil)
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, true, false, id, true)
	return messageHandler, nil
}

// ListenOnceForDestination Will listen for a single ResponseDir message on the Channel before un-subscribing automatically.
func (bus *transportEventBus) ListenOnceForDestination(channelName string, destId *uuid.UUID) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	if destId == nil {
		return nil, fmt.Errorf("DestinationId cannot be nil")
	}
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, false, false, destId, true)
	return messageHandler, nil
}

// RequestOnce Send a request message with Payload and wait for and Handle a single response message.
// Returns MessageHandler or error if the Channel is unknown
func (bus *transportEventBus) RequestOnce(channelName string, payload interface{}) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	destId := checkForSuppliedId(nil)
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, true, false, destId, true)
	config := buildConfig(channelName, payload, destId)
	message := model.GenerateRequest(config)
	messageHandler.requestMessage = message
	return messageHandler, nil
}

// RequestOnceForDestination Send a request message with Payload and wait for and Handle a single response message for a targeted DestinationId
// Returns MessageHandler or error if the Channel is unknown
func (bus *transportEventBus) RequestOnceForDestination(
	channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error) {

	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	if destId == nil {
		return nil, fmt.Errorf("DestinationId cannot be nil")
	}
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, false, false, destId, true)
	config := buildConfig(channelName, payload, destId)
	message := model.GenerateRequest(config)
	messageHandler.requestMessage = message
	return messageHandler, nil
}

func getChannelFromManager(bus *transportEventBus, channelName string) (*Channel, error) {
	channelManager := bus.ChannelManager
	channel, err := channelManager.GetChannel(channelName)
	return channel, err
}

// RequestStream Send a request message with Payload and wait for and Handle all response messages.
// Returns MessageHandler or error if Channel is unknown
func (bus *transportEventBus) RequestStream(channelName string, payload interface{}) (MessageHandler, error) {
	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	id := checkForSuppliedId(nil)
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, true, false, id, false)
	config := buildConfig(channelName, payload, id)
	message := model.GenerateRequest(config)
	messageHandler.requestMessage = message
	return messageHandler, nil
}

// RequestStreamForDestination Send a request message with Payload and wait for and Handle all response messages with a supplied DestinationId
// Returns MessageHandler or error if Channel is unknown
func (bus *transportEventBus) RequestStreamForDestination(
	channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error) {

	channel, err := getChannelFromManager(bus, channelName)
	if err != nil {
		return nil, err
	}
	if destId == nil {
		return nil, fmt.Errorf("DestinationId cannot be nil")
	}
	messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, false, false, destId, false)
	config := buildConfig(channelName, payload, destId)
	message := model.GenerateRequest(config)
	messageHandler.requestMessage = message
	return messageHandler, nil
}

// ConnectBroker Connect to a message broker. If successful, you get a pointer to a Connection. If not, you will get an error.
func (bus *transportEventBus) ConnectBroker(config *bridge.BrokerConnectorConfig) (conn bridge.Connection, err error) {
	conn, err = bus.bc.Connect(config, enableLogging)
	if conn != nil {
		bus.brokerConnections[conn.GetId()] = conn
	}
	return
}

// Start a new Fabric Endpoint
func (bus *transportEventBus) StartFabricEndpoint(
	connectionListener stompserver.RawConnectionListener, config EndpointConfig) error {

	if bus.fabEndpoint != nil {
		return fmt.Errorf("unable to start: fabric endpoint is already running")
	}
	if configErr := config.validate(); configErr != nil {
		return configErr
	}

	// start the store sync service the first time a fabric endpoint
	// is started.
	bus.initStoreSync.Do(func() {
		bus.storeSyncService = newStoreSyncService(bus)
	})

	bus.fabEndpoint = newFabricEndpoint(bus, connectionListener, config)
	bus.fabEndpoint.Start()
	return nil
}

func (bus *transportEventBus) StopFabricEndpoint() error {
	fe := bus.fabEndpoint
	if fe == nil {
		return fmt.Errorf("unable to stop: fabric endpoint is not running")
	}
	bus.fabEndpoint = nil
	fe.Stop()
	return nil
}

func (bus *transportEventBus) CreateAsyncTransaction() BusTransaction {
	return newBusTransaction(bus, asyncTransaction)
}

func (bus *transportEventBus) CreateSyncTransaction() BusTransaction {
	return newBusTransaction(bus, syncTransaction)
}

func (bus *transportEventBus) SendMonitorEvent(
	evtType MonitorEventType, entityName string, payload interface{}) {

	bus.monitor.sendEvent(NewMonitorEvent(evtType, entityName, payload))
}

func (bus *transportEventBus) wrapMessageHandler(
	channel *Channel, direction model.Direction, ignoreId bool, allTraffic bool, destId *uuid.UUID,
	runOnce bool) *messageHandler {

	messageHandler := createMessageHandler(channel, destId, bus.ChannelManager)
	messageHandler.ignoreId = ignoreId

	if runOnce {
		messageHandler.invokeOnce = &sync.Once{}
	}

	errorHandler := func(err error) {
		if messageHandler.errorHandler != nil {
			if runOnce {
				messageHandler.invokeOnce.Do(func() {
					atomic.AddInt64(&messageHandler.runCount, 1)
					messageHandler.errorHandler(err)

					bus.GetChannelManager().UnsubscribeChannelHandler(
						channel.Name, messageHandler.subscriptionId)
				})
			} else {
				atomic.AddInt64(&messageHandler.runCount, 1)
				messageHandler.errorHandler(err)
			}
		}
	}
	successHandler := func(msg *model.Message) {
		if messageHandler.successHandler != nil {
			if runOnce {
				messageHandler.invokeOnce.Do(func() {
					atomic.AddInt64(&messageHandler.runCount, 1)
					messageHandler.successHandler(msg)

					bus.GetChannelManager().UnsubscribeChannelHandler(
						channel.Name, messageHandler.subscriptionId)
				})
			} else {
				atomic.AddInt64(&messageHandler.runCount, 1)
				messageHandler.successHandler(msg)
			}
		}
	}

	handlerWrapper := func(msg *model.Message) {
		dir := direction
		id := messageHandler.destination
		if allTraffic {
			if msg.Direction == model.ErrorDir {
				errorHandler(msg.Error)
			} else {
				successHandler(msg)
			}
		} else {
			if msg.Direction == dir {
				// if we're checking for specific traffic, check a DestinationId match is required.
				if !messageHandler.ignoreId &&
					(msg.DestinationId != nil && id != nil) && (id.ID() == msg.DestinationId.ID()) {
					successHandler(msg)
				}
				if messageHandler.ignoreId {
					successHandler(msg)
				}
			}
			if msg.Direction == model.ErrorDir {
				errorHandler(msg.Error)
			}
		}
	}

	messageHandler.wrapperFunction = handlerWrapper
	return messageHandler
}

func checkForSuppliedId(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		i := uuid.New()
		id = &i
	}
	return id
}

func sendMessageToChannel(channelObject *Channel, message *model.Message) {
	channelObject.Send(message)
}

func buildConfig(channelName string, payload interface{}, destinationId *uuid.UUID) *model.MessageConfig {
	config := new(model.MessageConfig)
	id := uuid.New()
	config.Id = &id
	config.DestinationId = destinationId
	config.Channel = channelName
	config.Payload = payload
	return config
}

func buildError(channelName string, err error, destinationId *uuid.UUID) *model.MessageConfig {
	config := new(model.MessageConfig)
	id := uuid.New()
	config.Id = &id
	config.DestinationId = destinationId
	config.Channel = channelName
	config.Err = err
	return config
}

func createMessageHandler(channel *Channel, destinationId *uuid.UUID, channelMgr ChannelManager) *messageHandler {
	messageHandler := new(messageHandler)
	messageHandler.channel = channel
	id := uuid.New()
	messageHandler.id = &id
	messageHandler.destination = destinationId
	messageHandler.channelManager = channelMgr
	return messageHandler
}

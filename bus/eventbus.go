// Copyright 2019 VMware Inc.

package bus

import (
    "go-bifrost/bridge"
    "go-bifrost/model"
    "go-bifrost/util"
    "fmt"
    "github.com/google/uuid"
    "sync"
    "sync/atomic"
)

// EventBus provides access to ChannelManager, simple message sending and simple API calls for handling
// messaging and error handling over channels on the bus.
type EventBus interface {
    GetId() *uuid.UUID
    GetChannelManager() ChannelManager
    SendRequestMessage(channelName string, payload interface{}, destinationId *uuid.UUID) error
    SendResponseMessage(channelName string, payload interface{}, destinationId *uuid.UUID) error
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
    ConnectBroker(config *bridge.BrokerConnectorConfig) (conn *bridge.Connection, err error)
    StartTCPService(address string) error
}

var once sync.Once
var busInstance EventBus

// Get a reference to the EventBus.
func GetBus() EventBus {
    once.Do(func() {
        bf := new(bifrostEventBus)
        bf.init()
        busInstance = bf
    })
    return busInstance
}

type bifrostEventBus struct {
    ChannelManager    ChannelManager
    Id                uuid.UUID
    monitor           *util.MonitorStream
    brokerConnections map[*uuid.UUID]*bridge.Connection
    bc                bridge.BrokerConnector
}

func (bus *bifrostEventBus) GetId() *uuid.UUID {
    return &bus.Id
}

func (bus *bifrostEventBus) init() {

    bus.Id = uuid.New()
    bus.ChannelManager = NewBusChannelManager(bus)
    bus.monitor = util.GetMonitor()
    bus.brokerConnections = make(map[*uuid.UUID]*bridge.Connection)
    bus.bc = bridge.NewBrokerConnector()
    fmt.Printf("🌈 Bifröst booted with Id [%s]\n", bus.Id.String())
}

// Get a pointer to the ChannelManager for managing Channels.
func (bus *bifrostEventBus) GetChannelManager() ChannelManager {
    return bus.ChannelManager
}

// Send a ResponseDir type (inbound) message on Channel, with supplied Payload.
// Throws error if the Channel does not exist.
func (bus *bifrostEventBus) SendResponseMessage(channelName string, payload interface{}, destId *uuid.UUID) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload, destId)
    message := model.GenerateResponse(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a RequestDir type message (outbound) message on Channel, with supplied Payload.
// Throws error if the Channel does not exist.
func (bus *bifrostEventBus) SendRequestMessage(channelName string, payload interface{}, destId *uuid.UUID) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload, destId)
    message := model.GenerateRequest(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a ErrorDir type message (outbound) message on Channel, with supplied error
// Throws error if the Channel does not exist.
func (bus *bifrostEventBus) SendErrorMessage(channelName string, err error, destId *uuid.UUID) error {
    channelObject, chanErr := bus.ChannelManager.GetChannel(channelName)
    if chanErr != nil {
        return err
    }
    config := buildError(channelName, err, destId)
    message := model.GenerateError(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Listen to stream of ResponseDir (inbound) messages on Channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenStream(channelName string) (MessageHandler, error) {
    channel, err := getChannelFromManager(bus, channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, true, false, nil, false)
    return messageHandler, nil
}

// Listen to stream of ResponseDir (inbound) messages on Channel for a specific DestinationId.
// Will keep on ticking until closed, returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenStreamForDestination(channelName string, destId *uuid.UUID) (MessageHandler, error) {
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

// Listen to a stream of RequestDir (outbound) messages on Channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenRequestStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenRequestStream(channelName string) (MessageHandler, error) {
    channel, err := getChannelFromManager(bus, channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, true, false, nil, false)
    return messageHandler, nil
}

// Listen to a stream of RequestDir (outbound) messages on Channel for a specific DestinationId.
// Will keep on ticking until closed, returns MessageHandler
//  // To close an open stream.
//  handler, Err := bus.ListenRequestStream("my-Channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenRequestStreamForDestination(
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

// Listen for a single RequestDir (outbound) messages on Channel. Handler is closed after a single event.
// Returns MessageHandler
func (bus *bifrostEventBus) ListenRequestOnce(channelName string) (MessageHandler, error) {
    channel, err := getChannelFromManager(bus, channelName)
    if err != nil {
        return nil, err
    }
    id := checkForSuppliedId(nil)
    messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, true, false, id, true)
    return messageHandler, nil
}

// Listen for a single RequestDir (outbound) messages on Channel with a specific DestinationId.
// Handler is closed after a single event, returns MessageHandler
func (bus *bifrostEventBus) ListenRequestOnceForDestination(
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

func (bus *bifrostEventBus) ListenFirehose(channelName string) (MessageHandler, error) {
    channel, err := getChannelFromManager(bus, channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, model.RequestDir, true, true, nil, false)
    return messageHandler, nil
}

// Will listen for a single ResponseDir message on the Channel before un-subscribing automatically.
func (bus *bifrostEventBus) ListenOnce(channelName string) (MessageHandler, error) {
    channel, err := getChannelFromManager(bus, channelName)
    if err != nil {
        return nil, err
    }
    id := checkForSuppliedId(nil)
    messageHandler := bus.wrapMessageHandler(channel, model.ResponseDir, true, false, id, true)
    return messageHandler, nil
}

// Will listen for a single ResponseDir message on the Channel before un-subscribing automatically.
func (bus *bifrostEventBus) ListenOnceForDestination(channelName string, destId *uuid.UUID) (MessageHandler, error) {
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

// Send a request message with Payload and wait for and Handle a single response message.
// Returns MessageHandler or error if the Channel is unknown
func (bus *bifrostEventBus) RequestOnce(channelName string, payload interface{}) (MessageHandler, error) {
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

// Send a request message with Payload and wait for and Handle a single response message for a targeted DestinationId
// Returns MessageHandler or error if the Channel is unknown
func (bus *bifrostEventBus) RequestOnceForDestination(
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

func getChannelFromManager(bus *bifrostEventBus, channelName string) (*Channel, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    return channel, err
}

// Send a request message with Payload and wait for and Handle all response messages.
// Returns MessageHandler or error if Channel is unknown
func (bus *bifrostEventBus) RequestStream(channelName string, payload interface{}) (MessageHandler, error) {
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

// Send a request message with Payload and wait for and Handle all response messages with a supplied DestinationId
// Returns MessageHandler or error if Channel is unknown
func (bus *bifrostEventBus) RequestStreamForDestination(
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

// Connect to a message broker. If successful, you get a pointer to a Connection. If not, you will get an error.
func (bus *bifrostEventBus) ConnectBroker(config *bridge.BrokerConnectorConfig) (conn *bridge.Connection, err error) {
    conn, err = bus.bc.Connect(config)
    if conn != nil {
        bus.brokerConnections[conn.Id] = conn
    }
    bus.ChannelManager.ListenToMonitor()
    return
}

// Start a new STOMP TCP service operating on the supplied port
func (bus *bifrostEventBus) StartTCPService(address string) error {
    return bus.bc.StartTCPServer(address)
}

func (bus *bifrostEventBus) wrapMessageHandler(
    channel *Channel, direction model.Direction, ignoreId bool, allTraffic bool, destId *uuid.UUID,
    runOnce bool) *messageHandler {

    messageHandler := createMessageHandler(channel, destId)
    messageHandler.ignoreId = ignoreId

    if (runOnce) {
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
    if message.Error != nil {
        defer util.GetMonitor().SendMonitorEventData(util.ChannelErrorEvt, channelObject.Name, message)
    } else {
        defer util.GetMonitor().SendMonitorEventData(util.ChannelMessageEvt, channelObject.Name, message)
    }
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

func createMessageHandler(channel *Channel, destinationId *uuid.UUID) *messageHandler {
    messageHandler := new(messageHandler)
    messageHandler.channel = channel
    id := uuid.New()
    messageHandler.id = &id
    messageHandler.destination = destinationId
    return messageHandler
}


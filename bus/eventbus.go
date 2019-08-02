// Copyright 2019 VMware Inc.

package bus

import (
    "fmt"
    "github.com/google/uuid"
    "sync"
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
    ListenFirehose(channelName string) (MessageHandler, error)
    ListenRequestStream(channelName string, destinationId *uuid.UUID) (MessageHandler, error)
    ListenRequestOnce(channelName string) (MessageHandler, error)
    ListenOnce(channelName string) (MessageHandler, error)
    RequestOnce(channelName string, payload interface{}, destinationId *uuid.UUID) (MessageHandler, error)
    RequestStream(channelName string, payload interface{}) (MessageHandler, error)
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
    ChannelManager ChannelManager
    Id             uuid.UUID
}

func (bus *bifrostEventBus) GetId() *uuid.UUID {
    return &bus.Id
}

func (bus *bifrostEventBus) init() {

    bus.Id = uuid.New()
    bus.ChannelManager = new(busChannelManager)
    bus.ChannelManager.Boot()
    fmt.Printf("🌈 Bifröst booted with id [%s]", bus.Id.String())
}

// Get a pointer to the ChannelManager for managing Channels.
func (bus *bifrostEventBus) GetChannelManager() ChannelManager {
    return bus.ChannelManager
}

// Send a Response type (inbound) message on channel, with supplied payload.
// Throws error if the channel does not exist.
func (bus *bifrostEventBus) SendResponseMessage(channelName string, payload interface{}, destId *uuid.UUID) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload, destId)
    message := generateResponse(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a Request type message (outbound) message on channel, with supplied payload.
// Throws error if the channel does not exist.
func (bus *bifrostEventBus) SendRequestMessage(channelName string, payload interface{}, destId *uuid.UUID) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload, destId)
    message := generateRequest(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a Error type message (outbound) message on channel, with supplied error
// Throws error if the channel does not exist.
func (bus *bifrostEventBus) SendErrorMessage(channelName string, err error, destId *uuid.UUID) error {
    channelObject, chanErr := bus.ChannelManager.GetChannel(channelName)
    if chanErr != nil {
        return err
    }

    config := buildError(channelName, err, destId)
    message := generateError(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Listen to stream of Response (inbound) messages on channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, err := bus.ListenStream("my-channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenStream(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }

    messageHandler := bus.wrapMessageHandler(channel, Response, false, false, nil)
    return messageHandler, nil
}

// Listen to a stream of Request (outbound) messages on channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, err := bus.ListenRequestStream("my-channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenRequestStream(channelName string, destId *uuid.UUID) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    destId = checkForSuppliedId(destId)
    messageHandler := bus.wrapMessageHandler(channel, Request, false, false, destId)
    return messageHandler, nil
}

// Listen for a single Request (outbound) messages on channel. Handler is closed after a single event.
// Returns MessageHandler
func (bus *bifrostEventBus) ListenRequestOnce(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    id := checkForSuppliedId(nil)
    messageHandler := bus.wrapMessageHandler(channel, Request, false, false, id)
    messageHandler.runOnce = true
    return messageHandler, nil
}

func (bus *bifrostEventBus) ListenFirehose(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, Request, true, true, nil)
    return messageHandler, nil
}

// Will listen for a single Response message on the channel before un-subscribing automatically.
func (bus *bifrostEventBus) ListenOnce(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    id := checkForSuppliedId(nil)
    messageHandler := bus.wrapMessageHandler(channel, Response, false, false, id)
    messageHandler.runOnce = true
    return messageHandler, nil
}

// Send a request message with payload and wait for and Handle a single response message.
// Returns MessageHandler or error if the channel is unknown
func (bus *bifrostEventBus) RequestOnce(channelName string, payload interface{}, destId *uuid.UUID) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    destId = checkForSuppliedId(destId)
    messageHandler := bus.wrapMessageHandler(channel, Response, false, false, destId)
    config := buildConfig(channelName, payload, destId)
    message := generateRequest(config)
    messageHandler.requestMessage = message
    messageHandler.runOnce = true
    return messageHandler, nil
}

// Send a request message with payload and wait for and Handle all response messages with that ID.
// Returns MessageHandler or error if channel is unknown
func (bus *bifrostEventBus) RequestStream(channelName string, payload interface{}) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    id := checkForSuppliedId(nil)
    messageHandler := bus.wrapMessageHandler(channel, Response, false, false, id)
    config := buildConfig(channelName, payload, id)
    message := generateRequest(config)
    messageHandler.requestMessage = message
    messageHandler.runOnce = false
    return messageHandler, nil
}

func checkForSuppliedId(id *uuid.UUID) *uuid.UUID {
    if id == nil {
        i := uuid.New()
        id = &i
    }
    return id
}

func checkHandlerHasRun(handler *messageHandler) bool {
    return handler.hasRun
}

func checkHandlerSingleRun(handler *messageHandler) bool {
    return handler.runOnce
}

func (bus *bifrostEventBus) wrapMessageHandler(channel *Channel, direction Direction, ignoreId bool, allTraffic bool, destId *uuid.UUID) *messageHandler {
    messageHandler := createMessageHandler(channel, destId)
    errorHandler := func(err error) {
        if messageHandler.errorHandler != nil {
            if checkHandlerSingleRun(messageHandler) {
                if !checkHandlerHasRun(messageHandler) {
                    messageHandler.hasRun = true
                    messageHandler.runCount++
                    messageHandler.errorHandler(err)
                }
            } else {
                messageHandler.errorHandler(err)
            }
        }
    }
    successHandler := func(msg *Message) {
        if messageHandler.successHandler != nil {
            if checkHandlerSingleRun(messageHandler) {
                if !checkHandlerHasRun(messageHandler) {
                    messageHandler.hasRun = true
                    messageHandler.runCount++
                    messageHandler.successHandler(msg)
                }
            } else {
                messageHandler.runCount++
                messageHandler.successHandler(msg)
            }
        }
    }
    handlerWrapper := func(msg *Message) {
        dir := direction
        id := messageHandler.destination
        if allTraffic {
            if msg.Direction == Error {
                errorHandler(msg.Error)
            } else {
                successHandler(msg)
            }
        } else {
            if msg.Direction == dir {

                // if we're checking for specific traffic, check a destination match is required.
                if !ignoreId && (msg.DestinationId != nil && id != nil) && (id.ID() == msg.DestinationId.ID()) {
                    successHandler(msg)
                }
            }
            if msg.Direction == Error {
                errorHandler(msg.Error)
            }
        }
    }

    messageHandler.wrapperFunction = handlerWrapper
    return messageHandler
}

func sendMessageToChannel(channelObject *Channel, message *Message) {
    channelObject.Send(message)
}

func buildConfig(channelName string, payload interface{}, destinationId *uuid.UUID) *messageConfig {
    config := new(messageConfig)
    id := uuid.New()
    config.id = &id
    config.destination = destinationId
    config.channel = channelName
    config.payload = payload
    return config
}

func buildError(channelName string, err error, destinationId *uuid.UUID) *messageConfig {
    config := new(messageConfig)
    id := uuid.New()
    config.id = &id
    config.destination = destinationId
    config.channel = channelName
    config.err = err
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

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
    SendRequestMessage(channelName string, payload interface{}, id *uuid.UUID) error
    SendResponseMessage(channelName string, payload interface{}, id *uuid.UUID) error
    SendErrorMessage(channelName string, err error, id *uuid.UUID) error
    ListenStream(channelName string) (MessageHandler, error)
    ListenFirehose(channelName string) (MessageHandler, error)
    ListenRequestStream(channelName string) (MessageHandler, error)
    ListenOnce(channelName string) (MessageHandler, error)
    RequestOnce(channelName string, payload interface{}) (MessageHandler, error)
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
func (bus *bifrostEventBus) SendResponseMessage(channelName string, payload interface{}, id *uuid.UUID) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload, id)
    message := generateResponse(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a Request type message (outbound) message on channel, with supplied payload.
// Throws error if the channel does not exist.
func (bus *bifrostEventBus) SendRequestMessage(channelName string, payload interface{}, id *uuid.UUID) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload, id)
    message := generateRequest(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a Error type message (outbound) message on channel, with supplied error
// Throws error if the channel does not exist.
func (bus *bifrostEventBus) SendErrorMessage(channelName string, err error, id *uuid.UUID) error {
    channelObject, chanErr := bus.ChannelManager.GetChannel(channelName)
    if chanErr != nil {
        return err
    }

    config := buildError(channelName, err)
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

    messageHandler := bus.wrapMessageHandler(channel, Response, false, false)
    return messageHandler, nil
}

// Listen to a stream of Request (outbound) messages on channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, err := bus.ListenRequestStream("my-channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *bifrostEventBus) ListenRequestStream(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, Request, false, false)
    return messageHandler, nil
}

func (bus *bifrostEventBus) ListenFirehose(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, Request, true, true)
    return messageHandler, nil
}

// Will listen for a single Response message on the channel before un-subscribing automatically.
func (bus *bifrostEventBus) ListenOnce(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, Response, false, false)
    messageHandler.runOnce = true
    return messageHandler, nil
}

// Send a request message with payload and wait for and Handle a single response message.
// Returns MessageHandler or error if the channel is unknown
func (bus *bifrostEventBus) RequestOnce(channelName string, payload interface{}) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }

    messageHandler := bus.wrapMessageHandler(channel, Response, false, false)
    config := buildConfig(channelName, payload, nil)
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

    messageHandler := bus.wrapMessageHandler(channel, Response, false, false)
    config := buildConfig(channelName, payload, nil)
    message := generateRequest(config)
    messageHandler.requestMessage = message
    messageHandler.runOnce = false
    return messageHandler, nil
}

func checkHandlerHasRun(handler *messageHandler) bool {
    return handler.hasRun
}

func checkHandlerSingleRun(handler *messageHandler) bool {
    return handler.runOnce
}

func (bus *bifrostEventBus) wrapMessageHandler(channel *Channel, direction Direction, ignoreId bool, allTraffic bool) *messageHandler {
    messageHandler := createMessageHandler(channel)
    errorHandler := func(err error) {
        if messageHandler.errorHandler != nil {
            if checkHandlerSingleRun(messageHandler) {
                if !checkHandlerHasRun(messageHandler) {
                    messageHandler.errorHandler(err)
                    messageHandler.hasRun = true
                    messageHandler.runCount++
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
                    messageHandler.successHandler(msg)
                    messageHandler.hasRun = true
                    messageHandler.runCount++
                }
            } else {
                messageHandler.successHandler(msg)
                messageHandler.runCount++
            }
        }
    }
    handlerWrapper := func(msg *Message) {
        if allTraffic {
            if msg.Direction == Error {
                errorHandler(msg.Error)
            } else {
                successHandler(msg)
            }
        } else {
            if msg.Direction == direction {
                if !ignoreId && messageHandler.id.ID() == msg.Id.ID() {
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

func buildConfig(channelName string, payload interface{}, msgId *uuid.UUID) *messageConfig {
    config := new(messageConfig)
    var id *uuid.UUID
    if msgId == nil {
        i := uuid.New()
        id = &i
    }
    id = msgId
    config.id = id
    config.channel = channelName
    config.payload = payload
    return config
}

func buildError(channelName string, err error) *messageConfig {
    config := new(messageConfig)
    id := uuid.New()
    config.id = &id
    config.channel = channelName
    config.err = err
    return config
}

func createMessageHandler(channel *Channel) *messageHandler {
    messageHandler := new(messageHandler)
    messageHandler.channel = channel
    return messageHandler
}

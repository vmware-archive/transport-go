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
    SendRequestMessage(channelName string, payload interface{}) error
    SendResponseMessage(channelName string, payload interface{}) error
    SendErrorMessage(channelName string, err error) error
    ListenStream(channelName string) (MessageHandler, error)
    ListenRequestStream(channelName string) (MessageHandler, error)
    ListenOnce(channelName string) (MessageHandler, error)
}

// Signature used for all functions used on bus stream APIs to handle messages.
type MessageHandlerFunction func(*Message)

// Signature used for all functions used on bus stream APIs to handle errors.
type MessageErrorFunction func(error)

type MessageHandler interface {
    handle(successHandler MessageHandlerFunction, errorHandler MessageErrorFunction)
    //close()
}

type messageHandler struct {
    id             *uuid.UUID
    eventCount     int64
    closed         bool
    channel        *Channel
    successHandler MessageHandlerFunction
    errorHandler   MessageErrorFunction
}

var once sync.Once
var busInstance EventBus

// Get a reference to the EventBus.
func GetBus() EventBus {
    once.Do(func() {
        bf := new(BifrostEventBus)
        bf.init()
        busInstance = bf
    })
    return busInstance
}

type BifrostEventBus struct {
    ChannelManager ChannelManager
    Id             uuid.UUID
}

func (bus *BifrostEventBus) GetId() *uuid.UUID {
    return &bus.Id
}

func (bus *BifrostEventBus) init() {

    bus.Id = uuid.New()
    bus.ChannelManager = new(busChannelManager)
    bus.ChannelManager.Boot()
    fmt.Printf("🌈 Bifröst booted with id [%s]", bus.Id.String())
}

// Get a pointer to the ChannelManager for managing Channels.
func (bus *BifrostEventBus) GetChannelManager() ChannelManager {
    return bus.ChannelManager
}

// Send a Response type (inbound) message on channel, with supplied payload.
// Throws error if the channel does not exist.
func (bus *BifrostEventBus) SendResponseMessage(channelName string, payload interface{}) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload)
    message := generateResponse(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a Request type message (outbound) message on channel, with supplied payload.
// Throws error if the channel does not exist.
func (bus *BifrostEventBus) SendRequestMessage(channelName string, payload interface{}) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
        return err
    }
    config := buildConfig(channelName, payload)
    message := generateRequest(config)
    sendMessageToChannel(channelObject, message)
    return nil
}

// Send a Error type message (outbound) message on channel, with supplied error
// Throws error if the channel does not exist.
func (bus *BifrostEventBus) SendErrorMessage(channelName string, err error) error {
    channelObject, err := bus.ChannelManager.GetChannel(channelName)
    if err != nil {
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
func (bus *BifrostEventBus) ListenStream(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, _ := channelManager.GetChannel(channelName)
    messageHandler, messageHandlerWrapper := bus.wrapMessageHandler(channel, Response)

    id, err := channelManager.SubscribeChannelHandler(channelName, messageHandlerWrapper, false)
    if err != nil {
        return nil, err
    }

    messageHandler.id = id
    return messageHandler, nil
}

// Listen to a stream of Request (outbout) messages on channel. Will keep on ticking until closed.
// Returns MessageHandler
//  // To close an open stream.
//  handler, err := bus.ListenRequestStream("my-channel")
//  // ...
//  handler.close() // this will close the stream.
func (bus *BifrostEventBus) ListenRequestStream(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, _ := channelManager.GetChannel(channelName)
    messageHandler, messageHandlerWrapper := bus.wrapMessageHandler(channel, Request)

    id, err := channelManager.SubscribeChannelHandler(channelName, messageHandlerWrapper, false)
    if err != nil {
        return nil, err
    }

    messageHandler.id = id
    return messageHandler, nil
}

// Will listen for a single Response message on the channel before unsubscribing automatically.
func (bus *BifrostEventBus) ListenOnce(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, _ := channelManager.GetChannel(channelName)
    messageHandler, messageHandlerWrapper := bus.wrapMessageHandler(channel, Response)

    id, err := channelManager.SubscribeChannelHandler(channelName, messageHandlerWrapper, true)
    if err != nil {
        return nil, err
    }

    messageHandler.id = id
    return messageHandler, nil
}

func (bus *BifrostEventBus) wrapMessageHandler(channel *Channel, direction Direction) (*messageHandler, MessageHandlerFunction) {
    messageHandler := createMessageHandler(channel)
    errorHandler := func(err error) {
        if messageHandler.errorHandler != nil {
            messageHandler.errorHandler(err)
        }
    }
    successHandler := func(msg *Message) {
        if messageHandler.successHandler != nil {
            messageHandler.successHandler(msg)
        }
    }
    handlerWrapper := func(msg *Message) {
        if msg.Direction == direction {
            successHandler(msg)
        }
        if msg.Direction == Error {
            errorHandler(msg.Error)
        }
    }
    return messageHandler, handlerWrapper
}

func sendMessageToChannel(channelObject *Channel, message *Message) {
    channelObject.Send(message)
    channelObject.wg.Wait()
}

func buildConfig(channelName string, payload interface{}) *messageConfig {
    config := new(messageConfig)
    id := uuid.New()
    config.id = &id
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

func (msgHandler *messageHandler) handle(successHandler MessageHandlerFunction, errorHandler MessageErrorFunction) {
    msgHandler.successHandler = successHandler
    msgHandler.errorHandler = errorHandler
}

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
    //RequestStream(channelName string, payload interface{}) (MessageHandler, error)
}

// Signature used for all functions used on bus stream APIs to Handle messages.
type MessageHandlerFunction func(*Message)

// Signature used for all functions used on bus stream APIs to Handle errors.
type MessageErrorFunction func(error)

type MessageHandler interface {
    GetId() *uuid.UUID
    Handle(successHandler MessageHandlerFunction, errorHandler MessageErrorFunction)
    Fire() error
    //close()
}

type messageHandler struct {
    id              *uuid.UUID
    eventCount      int64
    closed          bool
    channel         *Channel
    requestMessage  *Message
    runOnce         bool
    ignoreId        bool
    wrapperFunction MessageHandlerFunction
    successHandler  MessageHandlerFunction
    errorHandler    MessageErrorFunction
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
func (bus *BifrostEventBus) SendResponseMessage(channelName string, payload interface{}, id *uuid.UUID) error {
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
func (bus *BifrostEventBus) SendRequestMessage(channelName string, payload interface{}, id *uuid.UUID) error {
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
func (bus *BifrostEventBus) SendErrorMessage(channelName string, err error, id *uuid.UUID) error {
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
func (bus *BifrostEventBus) ListenRequestStream(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, Request, false, false)
    return messageHandler, nil
}

func (bus *BifrostEventBus) ListenFirehose(channelName string) (MessageHandler, error) {
    channelManager := bus.ChannelManager
    channel, err := channelManager.GetChannel(channelName)
    if err != nil {
        return nil, err
    }
    messageHandler := bus.wrapMessageHandler(channel, Request, true, true)
    return messageHandler, nil
}

// Will listen for a single Response message on the channel before un-subscribing automatically.
func (bus *BifrostEventBus) ListenOnce(channelName string) (MessageHandler, error) {
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
// Returns MessageHandler
func (bus *BifrostEventBus) RequestOnce(channelName string, payload interface{}) (MessageHandler, error) {
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

func (bus *BifrostEventBus) wrapMessageHandler(channel *Channel, direction Direction, ignoreId bool, allTraffic bool) *messageHandler {
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
        if allTraffic {
            if msg.Direction == Error {
                errorHandler(msg.Error)
            } else {
                successHandler(msg)
            }
        }
        if msg.Direction == direction {
            if !ignoreId && messageHandler.id.ID() == msg.Id.ID() {
                successHandler(msg)
            }
        }
        if msg.Direction == Error {
            errorHandler(msg.Error)
        }
    }

    messageHandler.wrapperFunction = handlerWrapper
    return messageHandler
}

func sendMessageToChannel(channelObject *Channel, message *Message) {
    channelObject.Send(message)
    channelObject.wg.Wait()
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

func (msgHandler *messageHandler) Handle(successHandler MessageHandlerFunction, errorHandler MessageErrorFunction) {
    msgHandler.successHandler = successHandler
    msgHandler.errorHandler = errorHandler
    bus := GetBus().(*BifrostEventBus)
    channelManager := bus.GetChannelManager()

    id, err := channelManager.SubscribeChannelHandler(msgHandler.channel.Name, msgHandler.wrapperFunction, msgHandler.runOnce)
    if err != nil && errorHandler != nil {
        errorHandler(err)
    }
    msgHandler.id = id

    if msgHandler.requestMessage != nil {
        msgHandler.requestMessage.Id = id // align handler and message id.
    }
}

func (msgHandler *messageHandler) GetId() *uuid.UUID {
    return msgHandler.id
}

func (msgHandler *messageHandler) Fire() error {
    if msgHandler.requestMessage != nil {
        sendMessageToChannel(msgHandler.channel, msgHandler.requestMessage)
        return nil
    } else {
        return fmt.Errorf("nothing to fire, request is empty")
    }
}

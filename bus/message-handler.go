// Copyright 2019 VMware Inc.
package bus

import (
    "fmt"
    "github.com/google/uuid"
)

// Signature used for all functions used on bus stream APIs to Handle messages.
type MessageHandlerFunction func(*Message)

// Signature used for all functions used on bus stream APIs to Handle errors.
type MessageErrorFunction func(error)

// MessageHandler provides access to the ID the handler is listening for from all messages
// It also provides a Handle method that accepts a success and error function as handlers.
// The Fire method will fire the message queued when using RequestOnce or RequestStream
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
    hasRun          bool
    runCount        int
    ignoreId        bool
    wrapperFunction MessageHandlerFunction
    successHandler  MessageHandlerFunction
    errorHandler    MessageErrorFunction
}

func (msgHandler *messageHandler) Handle(successHandler MessageHandlerFunction, errorHandler MessageErrorFunction) {
    msgHandler.successHandler = successHandler
    msgHandler.errorHandler = errorHandler
    bus := GetBus().(*bifrostEventBus)
    channelManager := bus.GetChannelManager()

    id, _ := channelManager.SubscribeChannelHandler(msgHandler.channel.Name, msgHandler.wrapperFunction, msgHandler.runOnce)
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
        msgHandler.channel.wg.Wait()
        return nil
    } else {
        return fmt.Errorf("nothing to fire, request is empty")
    }
}

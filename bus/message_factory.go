// Copyright 2019 VMware Inc.
package bus

import "github.com/google/uuid"

type messageConfig struct {
    id        *uuid.UUID
    channel   string
    payload   interface{}
    direction Direction
    err       error
}

func checkId(msgConfig *messageConfig) {
    if msgConfig.id == nil {
        id := uuid.New()
        msgConfig.id = &id
    }
}

func generateRequest(msgConfig *messageConfig) *Message {
    checkId(msgConfig)
    return &Message{
        Id:        msgConfig.id,
        Channel:   msgConfig.channel,
        Payload:   msgConfig.payload,
        Direction: Request}
}

func generateResponse(msgConfig *messageConfig) *Message {
    checkId(msgConfig)
    return &Message{
        Id:        msgConfig.id,
        Channel:   msgConfig.channel,
        Payload:   msgConfig.payload,
        Direction: Response}
}

func generateError(msgConfig *messageConfig) *Message {
    checkId(msgConfig)
    return &Message{
        Id:        msgConfig.id,
        Channel:   msgConfig.channel,
        Error:     msgConfig.err,
        Direction: Error}
}

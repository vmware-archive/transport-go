// Copyright 2019 VMware Inc.
package bus

import "github.com/google/uuid"

type MessageConfig struct {
    Id          *uuid.UUID
    Destination *uuid.UUID
    Channel     string
    Payload     interface{}
    Direction   Direction
    Err         error
}

func checkId(msgConfig *MessageConfig) {
    if msgConfig.Id == nil {
        id := uuid.New()
        msgConfig.Id = &id
    }
}

func GenerateRequest(msgConfig *MessageConfig) *Message {
    checkId(msgConfig)
    return &Message{
        Id:            msgConfig.Id,
        Channel:       msgConfig.Channel,
        DestinationId: msgConfig.Destination,
        Payload:       msgConfig.Payload,
        Direction:     Request}
}

func GenerateResponse(msgConfig *MessageConfig) *Message {
    checkId(msgConfig)
    return &Message{
        Id:            msgConfig.Id,
        Channel:       msgConfig.Channel,
        DestinationId: msgConfig.Destination,
        Payload:       msgConfig.Payload,
        Direction:     Response}
}

func GenerateError(msgConfig *MessageConfig) *Message {
    checkId(msgConfig)
    return &Message{
        Id:            msgConfig.Id,
        Channel:       msgConfig.Channel,
        DestinationId: msgConfig.Destination,
        Error:         msgConfig.Err,
        Direction:     Error}
}

// Copyright 2019 VMware Inc.
package model

import "github.com/google/uuid"

// Direction int defining which way messages are travelling on a Channel.
type Direction int

const (
    RequestDir  Direction = 0
    ResponseDir Direction = 1
    ErrorDir    Direction = 2
)

// A Message is the encapsulation of the event sent on the bus.
// It holds a Direction, errors, a Payload and more.
type Message struct {
    Id            *uuid.UUID      `json:"id"`            // message identifier
    DestinationId *uuid.UUID      `json:"destinationId"` // destinationId (targeted recipient)
    Channel       string          `json:"channel"`       // reference to channel message was sent on.
    Destination   string          `json:"channel"`       // destination message was sent to (if galactic)
    Payload       interface{}     `json:"payload"`
    Error         error           `json:"error"`
    Direction     Direction       `json:"direction"`
    Headers       []MessageHeader `json:"headers"`
}

// A Message header can contain any meta data.
type MessageHeader struct {
    Label string
    Value string
}

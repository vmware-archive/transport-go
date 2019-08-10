// Copyright 2019 VMware Inc.
package bus

import "github.com/google/uuid"

// Direction int defining which way messages are travelling on a Channel.
type Direction int

const (
    Request  Direction = 0
    Response Direction = 1
    Error    Direction = 2
)

// A Message is the encapsulation of the event sent on the bus.
// It holds a Direction, errors, a Payload and more.
type Message struct {
    Id            *uuid.UUID      `json:"Id"`
    DestinationId *uuid.UUID      `json:"destinationId"`
    Channel       string          `json:"Channel"`
    Payload       interface{}     `json:"Payload"`
    Error         error           `json:"error"`
    Direction     Direction       `json:"Direction"`
    Headers       []MessageHeader `json:"headers"`
}

// A Message header can contain any meta data.
type MessageHeader struct {
    Label string
    Value string
}

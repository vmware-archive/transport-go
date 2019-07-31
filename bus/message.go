// Copyright 2019 VMware Inc.
package bus

import "github.com/google/uuid"

// Direction int defining which way messages are travelling on a channel.
type Direction int

const (
    Request Direction = 0
    Response Direction = 1
    Error Direction = 2
)

// A Message is the encapsulation of the event sent on the bus.
// It holds a direction, errors, a payload and more.
type Message struct {
    Id          *uuid.UUID      `json:"id"`
    Channel     string          `json:"channel"`
    Payload     interface{}     `json:"payload"`
    Error       error           `json:"error"`
    Direction   Direction       `json:"direction"`
    Headers     []MessageHeader `json:"headers"`
}

// A Message header can contain any meta data.
type MessageHeader struct {
    Label       string
    Value       string
}

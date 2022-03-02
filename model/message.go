// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

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
	Destination   string          `json:"destination"`   // destination message was sent to (if galactic)
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

// CastPayloadToType converts the raw interface{} typed Payload into the
// specified object passed as an argument.
func (m *Message) CastPayloadToType(typ interface{}) error {
	var unwrappedResponse Response

	// assert pointer type
	typVal := reflect.ValueOf(typ)
	if typVal.Kind() != reflect.Ptr {
		return fmt.Errorf("CastPayloadToType: invalid argument. argument should be the address of an object")
	}

	// nil-check
	if typVal.IsNil() {
		return fmt.Errorf("CastPayloadToType: cannot cast to nil")
	}

	// if message.Payload is already of *Response type, handle it here.
	if resp, ok := m.Payload.(*Response); ok {
		return decodeResponsePaylod(resp, typ)
	}

	// otherwise, unmrashal message.Payload into Response, then decode response.Payload
	if err := json.Unmarshal(m.Payload.([]byte), &unwrappedResponse); err != nil {
		return fmt.Errorf("CastPayloadToType: failed to unmarshal payload %v: %w", m.Payload, err)
	}

	return decodeResponsePaylod(&unwrappedResponse, typ)
}

// decodeResponsePaylod tries to unpack Response.Payload into typ.
func decodeResponsePaylod(resp *Response, typ interface{}) error {
	if resp.Error {
		return errors.New(resp.ErrorMessage)
	}
	return mapstructure.Decode(resp.Payload, typ)
}

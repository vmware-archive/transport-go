package model

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessage_CastPayloadToType_HappyPath(t *testing.T) {
	// arrange
	msg := getNewTestMessage()
	var dest Request

	// act
	err := msg.CastPayloadToType(&dest)

	// assert
	assert.Nil(t, err)
	assert.EqualValues(t, "dummy-value-coming-through", dest.Request)
}

func TestMessage_CastPayloadToType_BadPayload(t *testing.T) {
	// arrange
	msg := getNewTestMessage()
	pb := msg.Payload.([]byte)
	pb = append([]byte("random_bytes"), pb...)
	msg.Payload = pb
	var dest Request

	// act
	err := msg.CastPayloadToType(&dest)

	// assert
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal payload")
	assert.NotEqual(t, "dummy-value-coming-through", dest.Request)
}

func TestMessage_CastPayloadToType_NonPointer(t *testing.T) {
	// arrange
	msg := getNewTestMessage()
	var dest Request

	// act
	err := msg.CastPayloadToType(dest)

	// assert
	assert.NotNil(t, err)
	assert.NotEqual(t, "dummy-value-coming-through", dest.Request)
}

func TestMessage_CastPayloadToType_NilPointer(t *testing.T) {
	// arrange
	msg := getNewTestMessage()
	var dest *Request

	// act
	err := msg.CastPayloadToType(dest)

	// assert
	assert.NotNil(t, err)
	assert.Nil(t, dest)
}

func getNewTestMessage() *Message {
	rspPayload := &Response{
		Id:      &uuid.UUID{},
		Payload: Request{Request: "dummy-value-coming-through"},
	}

	jsonEncoded, _ := json.Marshal(rspPayload)
	return &Message{
		Id:      &uuid.UUID{},
		Channel: "test",
		Payload: jsonEncoded,
	}
}

package model

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"reflect"
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

func TestMessage_CastPayloadToType_UnmarshalledResponse_ByteSlice(t *testing.T) {
	// arrange
	msg := getUnmarshalledResponseMessage([]byte("I am a teapot"))
	dest := make([]byte, 0)

	// act
	err := msg.CastPayloadToType(&dest)

	// assert
	assert.Nil(t, err)
	assert.EqualValues(t, []byte("I am a teapot"), dest)
}

func TestMessage_CastPayloadToType_UnmarshalledResponse_Map(t *testing.T) {
	// arrange
	msg := getUnmarshalledResponseMessage(map[string]interface{}{"418": "I am a teapot"})
	dest := make(map[string]string)

	// act
	err := msg.CastPayloadToType(&dest)
	val, keyFound := dest["418"]

	// assert
	assert.Nil(t, err)
	assert.True(t, keyFound)
	assert.EqualValues(t, "I am a teapot", val)
}

func TestMessage_CastPayloadToType_ErrorResponse(t *testing.T) {
	// arrange
	msg := getErrorResponseMessage()
	dest := make([]byte, 0)

	// act
	err := msg.CastPayloadToType(&dest)

	// assert
	assert.NotNil(t, err)
	assert.EqualValues(t, "Bad Request", err.Error())
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

func getUnmarshalledResponseMessage(payload interface{}) *Message {
	rspPayload := &Response{
		Id:      &uuid.UUID{},
		Payload: payload,
	}

	return &Message{
		Id:      &uuid.UUID{},
		Channel: "test",
		Payload: reflect.ValueOf(rspPayload).Interface(),
	}
}

func getErrorResponseMessage() *Message {
	rspPayload := &Response{
		Id:           &uuid.UUID{},
		Error:        true,
		ErrorCode:    400,
		ErrorMessage: "Bad Request",
	}
	return &Message{
		Id:      &uuid.UUID{},
		Channel: "test",
		Payload: reflect.ValueOf(rspPayload).Interface(),
	}
}

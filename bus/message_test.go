// Copyright 2019 VMware Inc.
package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestMessageModel(t *testing.T) {
    id := uuid.New()
    var message = &Message{
        Id:        &id,
        Payload:   "A new message",
        Channel:   "123",
        Direction: Request}
    assert.Equal(t, "A new message", message.Payload)
    assert.Equal(t, Request, message.Direction, )
    assert.Equal(t, message.Channel, "123")
}

func TestMessageModel_ConfigId(t *testing.T) {
    config := new(MessageConfig)
    assert.Nil(t, config.Id)
    checkId(config)
    assert.NotNil(t, config.Id)
}

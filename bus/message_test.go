// Copyright 2019 VMware Inc.
package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestMessageModel(t *testing.T) {
    var message = &Message{
        Id:        uuid.New(),
        Payload:   "A new message",
        Channel:   "123",
        Direction: Request}
    assert.Equal(t, "A new message", message.Payload)
    assert.Equal(t, Request, message.Direction, )
    assert.Equal(t, message.Channel, "123")
}

func TestMessageModel_ConfigId(t *testing.T) {
    config := new(messageConfig)
    assert.Nil(t, config.id)
    checkId(config)
    assert.NotNil(t, config.id)
}

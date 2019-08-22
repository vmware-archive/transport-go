// Copyright 2019 VMware Inc.
package bus

import (
    "go-bifrost/model"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestMessageModel(t *testing.T) {
    id := uuid.New()
    var message = &model.Message{
        Id:        &id,
        Payload:   "A new message",
        Channel:   "123",
        Direction: model.RequestDir}
    assert.Equal(t, "A new message", message.Payload)
    assert.Equal(t, model.RequestDir, message.Direction, )
    assert.Equal(t, message.Channel, "123")
}


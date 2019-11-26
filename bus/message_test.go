// Copyright 2019 VMware Inc.
package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
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


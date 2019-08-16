// Copyright 2019 VMware Inc.
package util

import (
    "bifrost/model"
    "github.com/stretchr/testify/assert"
    "testing"
)

var m = GetMonitor()

func TestMonitorStream_SendMonitorEvent(t *testing.T) {

    done := make(chan bool)

    var listenChannelCreate = func() {
        evt := <-m.Stream
        assert.Equal(t, ChannelCreatedEvt, evt.EventType)
        assert.Equal(t, "happy-baby-melody", evt.Channel)
        done <- true
    }

    go listenChannelCreate()

    m.SendMonitorEvent(ChannelCreatedEvt, "happy-baby-melody")
    <-done
}

func TestMonitorStream_SendMonitorEventData(t *testing.T) {

    done := make(chan bool)

    var listenChannelCreate = func() {
        evt := <-m.Stream
        assert.Equal(t, ChannelCreatedEvt, evt.EventType)
        assert.Equal(t, "happy-baby-melody", evt.Channel)
        assert.Equal(t, "cutie", evt.Message.Payload)
        done <- true
    }

    go listenChannelCreate()

    msg := &model.Message{Payload: "cutie"}
    m.SendMonitorEventData(ChannelCreatedEvt, "happy-baby-melody", msg)
    <-done
}

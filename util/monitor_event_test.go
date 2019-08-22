// Copyright 2019 VMware Inc.
package util

import (
    "go-bifrost/model"
    "github.com/stretchr/testify/assert"
    "testing"
)

var m = GetMonitor()

func TestMonitorStream_SendMonitorEvent(t *testing.T) {

   done := make(chan bool)

   var listenChannelCreate = func() {
       go m.SendMonitorEvent(ChannelCreatedEvt, "happy-baby-melody")
       evt := <-m.Stream
       assert.Equal(t, ChannelCreatedEvt, evt.EventType)
       assert.Equal(t, "happy-baby-melody", evt.Channel)
       done <- true
   }

   go listenChannelCreate()
   <-done
}

func TestMonitorStream_SendMonitorEventData(t *testing.T) {

   done := make(chan bool)

   var listenChannelCreate = func() {
       msg := &model.Message{Payload: "cutie"}
       go m.SendMonitorEventData(ChannelCreatedEvt, "happy-baby-melody", msg)

       evt := <-m.Stream
       assert.Equal(t, ChannelCreatedEvt, evt.EventType)
       assert.Equal(t, "happy-baby-melody", evt.Channel)
       assert.Equal(t, "cutie", evt.Message.Payload)
       done <- true
   }
   go listenChannelCreate()
   <-done
}

func TestMonitorStream_ResetMonitor(t *testing.T) {
    m := GetMonitor()
    n := GetMonitor()
    assert.Equal(t, m, n)
    q := ResetMonitor()
    assert.NotEqual(t, m, q)
}

func TestMonitorStream_NoListener(t *testing.T) {
    m := GetMonitor()
    m.SendMonitorEvent(ChannelCreatedEvt, "none")
    m.SendMonitorEventData(ChannelCreatedEvt, "none", nil)
}


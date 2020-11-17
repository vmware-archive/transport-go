// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
    "github.com/go-stomp/stomp/frame"
    "github.com/stretchr/testify/assert"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
    "log"
    "os"
    "sync"
    "testing"
)

func TestBridgeClient_Disconnect(t *testing.T) {

    bc := new(BridgeClient)
    e := bc.Disconnect()
    assert.NotNil(t, e)
}

func TestBridgeClient_handleCommands(t *testing.T) {
    d := "rainbow-land"
    bc := new(BridgeClient)
    i := make(chan *frame.Frame, 1)
    e := make(chan *model.Message, 1)

    l := log.New(os.Stderr, "WebSocket Client: ", 2)
    bc.logger = l
    bc.inboundChan = i
    bc.Subscriptions = make(map[string]*BridgeClientSub)
    s := &BridgeClientSub{E: e, Destination: d}
    bc.Subscriptions[d] = s

    go bc.handleIncomingSTOMPFrames()
    wg := sync.WaitGroup{}

    var sendError = func() {
        f := frame.New(frame.ERROR, frame.Destination, d)
        bc.inboundChan <- f
        wg.Done()
    }
    wg.Add(1)
    go sendError()
    wg.Wait()
    m := <- s.E
    assert.Error(t, m.Error)
}

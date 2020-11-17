// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
    "github.com/go-stomp/stomp/frame"
    "github.com/google/uuid"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
    "sync"
)

// BridgeClientSub is a client subscription that encapsulates message and error channels for a subscription
type BridgeClientSub struct {
    C           chan *model.Message // MESSAGE payloads
    E           chan *model.Message // ERROR payloads.
    Id          *uuid.UUID
    Destination string
    Client      *BridgeClient
    subscribed  bool
    lock        sync.RWMutex
}

// Send an UNSUBSCRIBE frame for subscription destination.
func (cs *BridgeClientSub) Unsubscribe() {
    cs.lock.Lock()
    cs.subscribed = false
    cs.lock.Unlock()
    unsubscribeFrame := frame.New(frame.UNSUBSCRIBE,
        frame.Id, cs.Id.String(),
        frame.Destination, cs.Destination)

    cs.Client.SendFrame(unsubscribeFrame)
}

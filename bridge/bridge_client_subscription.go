// Copyright 2019 VMware Inc.

package bridge

import (
    "bifrost/model"
    "github.com/go-stomp/stomp"
    "github.com/go-stomp/stomp/frame"
    "github.com/google/uuid"
)

// BridgeClientSub is a client subscription that encapsulates message and error channels for a subscription
type BridgeClientSub struct {
    C           chan *model.Message // MESSAGE payloads
    E           chan *model.Message // ERROR payloads.
    Id          *uuid.UUID
    Destination string
    Client      *BridgeClient
    subscribed  bool
}

// Send an UNSUBSCRIBE frame for subscription destination.
func (cs *BridgeClientSub) Unsubscribe() {

    unsubscribeFrame := frame.New(frame.UNSUBSCRIBE,
        frame.Id, cs.Id.String(),
        frame.Destination, cs.Destination,
        frame.Ack, stomp.AckAuto.String())

    cs.Client.SendFrame(unsubscribeFrame)
    cs.subscribed = false
}

// Copyright 2019 VMware Inc.

package bridge

import (
    "go-bifrost/model"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
)

// Subscription represents a subscription to a broker destination.
type Subscription struct {
    C           chan *model.Message // listen to this for incoming messages
    Id          *uuid.UUID
    Destination string              // Destination of where this message was sent.
    stompTCPSub *stomp.Subscription
    wsStompSub  *BridgeClientSub
}

// Unsubscribe from destination. All channels will be closed.
func (s *Subscription) Unsubscribe() error {

    // if we're using TCP
    if s.stompTCPSub != nil {
        go s.stompTCPSub.Unsubscribe() // local broker hangs, so lets make sure it is non blocking.
        close(s.C)
        return nil
    }

    // if we're using Websockets.
    if s.wsStompSub != nil {
        s.wsStompSub.Unsubscribe()
        close(s.C)
        return nil
    }
    return fmt.Errorf("cannot unsubscribe from destination %s, no connection", s.Destination)
}

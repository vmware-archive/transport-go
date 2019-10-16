// Copyright 2019 VMware Inc.

package bridge

import (
    "go-bifrost/model"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
)

type Subscription interface {
    GetId() *uuid.UUID
    GetMsgChannel() chan *model.Message
    GetDestination() string
    Unsubscribe() error
}

// Subscription represents a subscription to a broker destination.
type subscription struct {
    c           chan *model.Message // listen to this for incoming messages
    id          *uuid.UUID
    destination string              // Destination of where this message was sent.
    stompTCPSub *stomp.Subscription
    wsStompSub  *BridgeClientSub
}

func (s *subscription) GetId() *uuid.UUID {
    return s.id
}

func (s *subscription) GetMsgChannel() chan *model.Message {
    return s.c
}

func (s *subscription) GetDestination() string {
    return s.destination
}

// Unsubscribe from destination. All channels will be closed.
func (s *subscription) Unsubscribe() error {

    // if we're using TCP
    if s.stompTCPSub != nil {
        go s.stompTCPSub.Unsubscribe() // local broker hangs, so lets make sure it is non blocking.
        close(s.c)
        return nil
    }

    // if we're using Websockets.
    if s.wsStompSub != nil {
        s.wsStompSub.Unsubscribe()
        close(s.c)
        return nil
    }
    return fmt.Errorf("cannot unsubscribe from destination %s, no connection", s.destination)
}

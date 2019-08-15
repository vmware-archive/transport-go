package bridge

import (
    "bifrost/bus"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
)

// This is an abstraction to encapsulate subscriptions.
type Subscription struct {
    C           chan *bus.Message
    Id          *uuid.UUID
    Destination string
    stompTCPSub *stomp.Subscription
    wsStompSub  *BridgeClientSub
}

func (s *Subscription) Unsubscribe() error {

    if s.stompTCPSub != nil {

        go s.stompTCPSub.Unsubscribe() // local broker hangs, so lets make sure it is non blocking.
        close(s.C)
        return nil
    }

    if s.wsStompSub != nil {
        s.wsStompSub.Unsubscribe()
        close(s.C)
        return nil
    }

    return fmt.Errorf("cannot unsubscribe from destination %s, no connection", s.Destination)
}

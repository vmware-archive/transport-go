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

        err := s.stompTCPSub.Unsubscribe()
        close(s.C)
        return err
    }

    if s.wsStompSub != nil {
        s.wsStompSub.Unsubscribe()
        close(s.C)
        return nil
    }

    return fmt.Errorf("cannot unsubscribe from destination %s, no connection", s.Destination)
}

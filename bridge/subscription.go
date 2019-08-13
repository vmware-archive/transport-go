package bridge

import (
    "bifrost/bus"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
)

// This is an abstraction to encapsulate subscriptions.
type Subscription struct {
    C           chan *bus.Message
    Id          *uuid.UUID
    stompTCPSub *stomp.Subscription
    wsStompSub  *BridgeClientSub
}

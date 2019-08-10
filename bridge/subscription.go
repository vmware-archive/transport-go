package bridge

import (
    "bifrost/bus"
    "github.com/go-stomp/stomp"
    "github.com/google/uuid"
)

// This is an abstraction to encapsulate subscriptions.
type Subscription struct {
    Channel     *bus.Channel
    Id          *uuid.UUID
    StompSub    *stomp.Subscription
}

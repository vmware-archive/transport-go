// Copyright 2019 VMware Inc.

package bus

import (
    "github.com/google/uuid"
)

type channelEventHandler struct {
    callBackFunction MessageHandlerFunction
    runOnce          bool
    uuid             uuid.UUID
}

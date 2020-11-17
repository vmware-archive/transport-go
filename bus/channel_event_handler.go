// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
    "github.com/google/uuid"
)

type channelEventHandler struct {
    callBackFunction MessageHandlerFunction
    runOnce          bool
    runCount         int64
    uuid             *uuid.UUID
}

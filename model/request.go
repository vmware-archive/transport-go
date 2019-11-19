// Copyright 2019 VMware Inc.

package model

import "github.com/google/uuid"

type Request struct {
    Id                *uuid.UUID               `json:"id"`
    Created           int64                    `json:"created"`
    Version           int                      `json:"version"`
    Destination       string                   `json:"channel"`
    Payload           interface{}              `json:"payload"`
    Request           string                   `json:"request"`
    // Populated if the request was sent on a "private" channel and
    // indicates where to send back the Response.
    // A service should check this field and if not null copy it to the
    // Response.BrokerDestination field to ensure that the response will be sent
    // back on the correct the "private" channel.
    BrokerDestination *BrokerDestinationConfig `json:"-"`
}

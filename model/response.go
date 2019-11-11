// Copyright 2019 VMware Inc.
package model

import "github.com/google/uuid"

// ResponseDir represents a payload sent by a Fabric application.
type Response struct {
    Id                *uuid.UUID               `json:"id"`
    Created           int64                    `json:"created"`
    Version           int                      `json:"version"`
    Destination       string                   `json:"channel"`
    Payload           interface{}              `json:"payload"`
    Error             bool                     `json:"error"`
    ErrorCode         int                      `json:"errorCode"`
    ErrorMessage      string                   `json:"errorMessage"`
    // If populated the response will be send to a single client
    // on the specified destination topic.
    BrokerDestination *BrokerDestinationConfig `json:"-"`
}

// Used to specify the target user queue of the Response
type BrokerDestinationConfig struct {
    Destination string
    ConnectionId string
}
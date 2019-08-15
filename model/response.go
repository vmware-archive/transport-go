// Copyright 2019 VMware Inc.
package model

import "github.com/google/uuid"

// ResponseDir represents a payload sent by a Fabric application.
type Response struct {
    Id            *uuid.UUID      `json:"id"`
    Created       int64           `json:"created"`
    Version       int             `json:"version"`
    Destination   string          `json:"channel"`
    Payload       interface{}     `json:"payload"`
    Error         bool            `json:"error"`
    ErrorCode     int             `json:"errorCode"`
    ErrorMessage  string          `json:"errorMessage"`
}

// Copyright 2019 VMware Inc.

package model

import "github.com/google/uuid"

type Request struct {
    Id            *uuid.UUID      `json:"id"`
    Created       int64           `json:"created"`
    Version       int             `json:"version"`
    Destination   string          `json:"channel"`
    Payload       interface{}     `json:"payload"`
    Request       string          `json:"request"`
}

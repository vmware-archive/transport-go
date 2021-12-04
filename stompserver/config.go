// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package stompserver

import "strings"

type StompConfig interface {
	HeartBeat() int64
	AppDestinationPrefix() []string
	IsAppRequestDestination(destination string) bool
}

type stompConfig struct {
	heartbeat     int64
	appDestPrefix []string
}

func NewStompConfig(heartBeatMs int64, appDestinationPrefix []string) StompConfig {
	prefixes := make([]string, len(appDestinationPrefix))
	for i := 0; i < len(appDestinationPrefix); i++ {
		if appDestinationPrefix[i] != "" && !strings.HasSuffix(appDestinationPrefix[i], "/") {
			prefixes[i] = appDestinationPrefix[i] + "/"
		} else {
			prefixes[i] = appDestinationPrefix[i]
		}
	}

	return &stompConfig{
		heartbeat:     heartBeatMs,
		appDestPrefix: prefixes,
	}
}

func (c *stompConfig) HeartBeat() int64 {
	return c.heartbeat
}

func (c *stompConfig) AppDestinationPrefix() []string {
	return c.appDestPrefix
}

func (c *stompConfig) IsAppRequestDestination(destination string) bool {
	for _, prefix := range c.appDestPrefix {
		if prefix != "" && strings.HasPrefix(destination, prefix) {
			return true
		}
	}
	return false
}

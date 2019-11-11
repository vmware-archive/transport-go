// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package stompserver

const (
    notConnectedStompError       = stompErrorMessage("not connected")
    unexpectedStompCommandError  = stompErrorMessage("unexpected frame command")
    unsupportedStompCommandError = stompErrorMessage("unsupported command")
    unsupportedStompVersionError = stompErrorMessage("unsupported STOMP version")
    invalidSubscriptionError     = stompErrorMessage("invalid subscription")
    invalidFrameError            = stompErrorMessage("invalid frame")
    invalidHeaderError           = stompErrorMessage("invalid frame header")
)

type stompErrorMessage string

func (e stompErrorMessage) Error() string {
    return string(e)
}

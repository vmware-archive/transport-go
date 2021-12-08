// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package stompserver

const (
	notConnectedStompError       = stompErrorMessage("not connected")
	unexpectedStompCommandError  = stompErrorMessage("unexpected frame command")
	unsupportedStompCommandError = stompErrorMessage("unsupported command")
	unsupportedStompVersionError = stompErrorMessage("unsupported STOMP version")
	invalidSubscriptionError     = stompErrorMessage("invalid subscription")
	invalidFrameError            = stompErrorMessage("invalid frame")
	invalidHeaderError           = stompErrorMessage("invalid frame header")
	invalidSendDestinationError  = stompErrorMessage("invalid send destination")
)

type stompErrorMessage string

func (e stompErrorMessage) Error() string {
	return string(e)
}

// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package stompserver

import (
	"fmt"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/google/uuid"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type subscription struct {
	id          string
	destination string
}

type StompConn interface {
	// Return unique connection Id string
	GetId() string
	SendFrameToSubscription(f *frame.Frame, sub *subscription)
	Close()
}

const (
	maxHeartBeatDuration = time.Duration(999999999) * time.Millisecond
)

const (
	connecting int32 = iota
	connected
	closed
)

type stompConn struct {
	rawConnection    RawConnection
	state            int32
	version          stomp.Version
	inFrames         chan *frame.Frame
	outFrames        chan *frame.Frame
	readTimeoutMs    int64
	writeTimeout     time.Duration
	id               string
	events           chan *ConnEvent
	config           StompConfig
	subscriptions    map[string]*subscription
	currentMessageId uint64
	closeOnce        sync.Once
}

func NewStompConn(rawConnection RawConnection, config StompConfig, events chan *ConnEvent) StompConn {
	conn := &stompConn{
		rawConnection: rawConnection,
		state:         connecting,
		inFrames:      make(chan *frame.Frame, 32),
		outFrames:     make(chan *frame.Frame, 32),
		config:        config,
		id:            uuid.New().String(),
		events:        events,
		subscriptions: make(map[string]*subscription),
	}

	go conn.run()
	go conn.readInFrames()

	return conn
}

func (conn *stompConn) SendFrameToSubscription(f *frame.Frame, sub *subscription) {
	f.Header.Add(frame.Subscription, sub.id)
	conn.outFrames <- f
}

func (conn *stompConn) Close() {
	conn.closeOnce.Do(func() {
		atomic.StoreInt32(&conn.state, closed)
		conn.rawConnection.Close()

		conn.events <- &ConnEvent{
			ConnId:    conn.GetId(),
			eventType: ConnectionClosed,
			conn:      conn,
		}
	})
}

func (conn *stompConn) GetId() string {
	return conn.id
}

func (conn *stompConn) run() {
	defer conn.Close()

	var timerChannel <-chan time.Time
	var timer *time.Timer

	for {

		if atomic.LoadInt32(&conn.state) == closed {
			return
		}

		if timer == nil && conn.writeTimeout > 0 {
			timer = time.NewTimer(conn.writeTimeout)
			timerChannel = timer.C
		}

		select {
		case f, ok := <-conn.outFrames:
			if !ok {
				// close connection
				return
			}

			// reset heart-beat timer
			if timer != nil {
				timer.Stop()
				timer = nil
			}

			conn.populateMessageIdHeader(f)

			// write the frame to the client
			err := conn.rawConnection.WriteFrame(f)
			if err != nil || f.Command == frame.ERROR {
				return
			}

		case f, ok := <-conn.inFrames:
			if !ok {
				return
			}

			if err := conn.handleIncomingFrame(f); err != nil {
				conn.sendError(err)
				return
			}

		case _ = <-timerChannel:
			// write a heart-beat
			err := conn.rawConnection.WriteFrame(nil)
			if err != nil {
				return
			}
			if timer != nil {
				timer.Stop()
				timer = nil
			}
		}
	}
}

func (conn *stompConn) handleIncomingFrame(f *frame.Frame) error {
	switch f.Command {

	case frame.CONNECT, frame.STOMP:
		return conn.handleConnect(f)

	case frame.DISCONNECT:
		return conn.handleDisconnect(f)

	case frame.SEND:
		return conn.handleSend(f)

	case frame.SUBSCRIBE:
		return conn.handleSubscribe(f)

	case frame.UNSUBSCRIBE:
		return conn.handleUnsubscribe(f)
	}

	return unsupportedStompCommandError
}

// Returns true if the frame contains ANY of the specified
// headers
func containsHeader(f *frame.Frame, headers ...string) bool {
	for _, h := range headers {
		if _, ok := f.Header.Contains(h); ok {
			return true
		}
	}
	return false
}

func (conn *stompConn) handleConnect(f *frame.Frame) error {
	if atomic.LoadInt32(&conn.state) == connected {
		return unexpectedStompCommandError
	}

	if containsHeader(f, frame.Receipt) {
		return invalidHeaderError
	}

	var err error
	conn.version, err = determineVersion(f)
	if err != nil {
		log.Println("cannot determine version")
		return err
	}

	if conn.version == stomp.V10 {
		return unsupportedStompVersionError
	}

	cxDuration, cyDuration, err := getHeartBeat(f)
	if err != nil {
		log.Println("invalid heart-beat")
		return err
	}

	min := time.Duration(conn.config.HeartBeat()) * time.Millisecond
	if min > maxHeartBeatDuration {
		min = maxHeartBeatDuration
	}

	// apply a minimum heartbeat
	if cxDuration > 0 {
		if min == 0 || cxDuration < min {
			cxDuration = min
		}
	}
	if cyDuration > 0 {
		if min == 0 || cyDuration < min {
			cyDuration = min
		}
	}

	conn.writeTimeout = cyDuration

	cx, cy := int64(cxDuration/time.Millisecond), int64(cyDuration/time.Millisecond)
	atomic.StoreInt64(&conn.readTimeoutMs, cx)

	response := frame.New(frame.CONNECTED,
		frame.Version, string(conn.version),
		frame.Server, "stompServer/0.0.1",
		frame.HeartBeat, fmt.Sprintf("%d,%d", cy, cx))

	err = conn.rawConnection.WriteFrame(response)
	if err != nil {
		return err
	}

	atomic.StoreInt32(&conn.state, connected)

	conn.events <- &ConnEvent{
		ConnId:    conn.GetId(),
		eventType: ConnectionEstablished,
		conn:      conn,
	}

	return nil
}

func (conn *stompConn) handleDisconnect(f *frame.Frame) error {
	if atomic.LoadInt32(&conn.state) == connecting {
		return notConnectedStompError
	}

	conn.sendReceiptResponse(f)
	conn.Close()

	return nil
}

func (conn *stompConn) handleSubscribe(f *frame.Frame) error {
	switch atomic.LoadInt32(&conn.state) {
	case connecting:
		return notConnectedStompError
	case closed:
		return nil
	}

	subId, ok := f.Header.Contains(frame.Id)
	if !ok {
		return invalidSubscriptionError
	}

	dest, ok := f.Header.Contains(frame.Destination)
	if !ok {
		return invalidFrameError
	}

	if _, exists := conn.subscriptions[subId]; exists {
		// subscription already exists
		return nil
	}

	conn.subscriptions[subId] = &subscription{
		id:          subId,
		destination: dest,
	}

	conn.events <- &ConnEvent{
		ConnId:      conn.GetId(),
		eventType:   SubscribeToTopic,
		destination: dest,
		conn:        conn,
		sub:         conn.subscriptions[subId],
		frame:       f,
	}

	return nil
}

func (conn *stompConn) handleUnsubscribe(f *frame.Frame) error {
	switch atomic.LoadInt32(&conn.state) {
	case connecting:
		return notConnectedStompError
	case closed:
		return nil
	}

	id, ok := f.Header.Contains(frame.Id)
	if !ok {
		return invalidSubscriptionError
	}

	conn.sendReceiptResponse(f)

	sub, ok := conn.subscriptions[id]
	if !ok {
		// subscription already removed
		return nil
	}

	// remove the subscription
	delete(conn.subscriptions, id)

	conn.events <- &ConnEvent{
		ConnId:      conn.GetId(),
		eventType:   UnsubscribeFromTopic,
		conn:        conn,
		sub:         sub,
		destination: sub.destination,
	}

	return nil
}

func (conn *stompConn) handleSend(f *frame.Frame) error {
	switch atomic.LoadInt32(&conn.state) {
	case connecting:
		return notConnectedStompError
	case closed:
		return nil
	}

	// TODO: Remove if we start supporting transactions
	if containsHeader(f, frame.Transaction) {
		return unsupportedStompCommandError
	}

	// no destination triggers an error
	dest, ok := f.Header.Contains(frame.Destination)
	if !ok {
		return invalidFrameError
	}

	// reject SENDing directly to non-request channels by clients
	if !conn.config.IsAppRequestDestination(f.Header.Get(frame.Destination)) {
		return invalidSendDestinationError
	}

	err := conn.sendReceiptResponse(f)
	if err != nil {
		return err
	}

	f.Command = frame.MESSAGE
	conn.events <- &ConnEvent{
		ConnId:      conn.GetId(),
		eventType:   IncomingMessage,
		destination: dest,
		frame:       f,
		conn:        conn,
	}

	return nil
}

func (conn *stompConn) sendReceiptResponse(f *frame.Frame) error {
	if receipt, ok := f.Header.Contains(frame.Receipt); ok {
		f.Header.Del(frame.Receipt)
		return conn.rawConnection.WriteFrame(frame.New(frame.RECEIPT, frame.ReceiptId, receipt))
	}
	return nil
}

func (conn *stompConn) readInFrames() {
	defer func() {
		close(conn.inFrames)
	}()

	infiniteTimeout := time.Time{}
	var readTimeoutMs int64 = 0
	for {
		readTimeoutMs = atomic.LoadInt64(&conn.readTimeoutMs)
		if readTimeoutMs > 0 {
			conn.rawConnection.SetReadDeadline(time.Now().Add(
				time.Duration(readTimeoutMs) * time.Millisecond))
		} else {
			conn.rawConnection.SetReadDeadline(infiniteTimeout)
		}

		f, err := conn.rawConnection.ReadFrame()
		if err != nil {
			return
		}

		if f == nil {
			// heartbeat frame
			continue
		}

		conn.inFrames <- f
	}
}

func determineVersion(f *frame.Frame) (stomp.Version, error) {
	if acceptVersion, ok := f.Header.Contains(frame.AcceptVersion); ok {
		versions := strings.Split(acceptVersion, ",")
		for _, supportedVersion := range []stomp.Version{stomp.V12, stomp.V11, stomp.V10} {
			for _, v := range versions {
				if v == supportedVersion.String() {
					// return the highest supported version
					return supportedVersion, nil
				}
			}
		}
	} else {
		return stomp.V10, nil
	}

	var emptyVersion stomp.Version
	return emptyVersion, unsupportedStompVersionError
}

func getHeartBeat(f *frame.Frame) (cx, cy time.Duration, err error) {
	if heartBeat, ok := f.Header.Contains(frame.HeartBeat); ok {
		return frame.ParseHeartBeat(heartBeat)
	}
	return 0, 0, nil
}

func (conn *stompConn) sendError(err error) {
	errorFrame := frame.New(frame.ERROR,
		frame.Message, err.Error())

	conn.rawConnection.WriteFrame(errorFrame)
}

func (conn *stompConn) populateMessageIdHeader(f *frame.Frame) {
	if f.Command == frame.MESSAGE {
		// allocate the value of message-id for this frame
		conn.currentMessageId++
		messageId := strconv.FormatUint(conn.currentMessageId, 10)
		f.Header.Set(frame.MessageId, messageId)
		// remove the Ack header (if any) as we don't support those
		f.Header.Del(frame.Ack)
	}
}

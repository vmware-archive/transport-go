// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package stompserver

import (
    "errors"
    "fmt"
    "github.com/go-stomp/stomp/frame"
    "github.com/stretchr/testify/assert"
    "sync"
    "testing"
    "time"
)

type MockRawConnection struct {
    connected bool
    incomingFrames chan interface{}
    lock sync.Mutex
    currentDeadline time.Time
    sentFrames []*frame.Frame
    nextWriteErr error
    nextReadErr error
    writeWg *sync.WaitGroup
}

func NewMockRawConnection() *MockRawConnection {
    return &MockRawConnection{
        connected: true,
        incomingFrames: make(chan interface{}),
        sentFrames: []*frame.Frame{},
        nextWriteErr: nil,
    }
}

func (con *MockRawConnection) ReadFrame() (*frame.Frame, error) {
    obj := <- con.incomingFrames

    if obj == nil {
        // heart-beat
        return nil, nil
    }

    f, ok := obj.(*frame.Frame)
    if ok {
        return f, nil
    }

    return nil, obj.(error)
}

func (con *MockRawConnection) WriteFrame(frame *frame.Frame) error {
    defer func() { con.nextWriteErr = nil}()
    if con.nextWriteErr != nil {
        return con.nextWriteErr
    }

    con.lock.Lock()
    con.sentFrames = append(con.sentFrames, frame)
    if con.writeWg != nil {
        con.writeWg.Done()
    }
    con.lock.Unlock()
    return nil
}

func (con *MockRawConnection) LastSentFrame() *frame.Frame {
    return con.sentFrames[len(con.sentFrames) - 1]
}

func (con *MockRawConnection) SetReadDeadline(t time.Time) {
    con.lock.Lock()
    con.currentDeadline = t
    con.lock.Unlock()
}

func (con *MockRawConnection) getCurrentReadDeadline() time.Time {
    con.lock.Lock()
    defer con.lock.Unlock()
    return con.currentDeadline
}

func (con *MockRawConnection) Close() error {
    con.connected = false
    return nil
}

func (con *MockRawConnection) SendConnectFrame() {
    con.incomingFrames <- frame.New(
        frame.CONNECT,
        frame.AcceptVersion, "1.2")
}

func getTestStompConn(conf StompConfig, events chan *ConnEvent) (*stompConn, *MockRawConnection, chan *ConnEvent) {
    if events == nil {
        events = make(chan *ConnEvent, 1000)
    }

    rawConn := NewMockRawConnection()
    return NewStompConn(rawConn, conf, events).(*stompConn), rawConn, events
}

func TestStompConn_Connect(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.NotNil(t, stompConn.GetId())

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New(frame.CONNECT, frame.AcceptVersion, "1.0,1.2,1.1,1.3")

    e := <- events

    assert.Equal(t, e.eventType, ConnectionEstablished)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED,
            frame.Version, "1.2",
            frame.HeartBeat, "0,0",
            frame.Server, "stompServer/0.0.1"), true)

    assert.Equal(t, stompConn.state, connected)
}

func TestStompConn_ConnectStomp10(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New(frame.CONNECT, frame.AcceptVersion, "1.0")

    e := <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)

    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, unsupportedStompVersionError.Error()), true)

    assert.Equal(t, stompConn.state, closed)
    assert.Equal(t, rawConn.connected, false)
}

func TestStompConn_ConnectInvalidStompVersion(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New(frame.CONNECT, frame.AcceptVersion, "5.0")

    e := <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)

    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, unsupportedStompVersionError.Error()), true)

    assert.Equal(t, stompConn.state, closed)
    assert.Equal(t, rawConn.connected, false)
}

func TestStompConn_ConnectWithReceiptHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New(frame.CONNECT,
        frame.AcceptVersion, "1.2",
        frame.Receipt, "receipt-id")

    e := <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)

    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, invalidHeaderError.Error()), true)

    assert.Equal(t, stompConn.state, closed)
    assert.Equal(t, rawConn.connected, false)
}

func TestStompConn_ConnectMissingStompVersionHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New(frame.CONNECT)

    e := <- events
    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)

    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, unsupportedStompVersionError.Error()), true)

    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_ConnectInvalidHeartbeatHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New(frame.CONNECT,
        frame.AcceptVersion, "1.2",
        frame.HeartBeat, "12,asd")

    e := <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)

    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, frame.ErrInvalidHeartBeat.Error()), true)

    assert.Equal(t, stompConn.state, closed)
    assert.Equal(t, rawConn.connected, false)
}

func TestStompConn_InvalidStompCommand(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    assert.Equal(t, stompConn.state, connecting)

    rawConn.incomingFrames <- frame.New("invalid-stomp-command",
        frame.AcceptVersion, "1.2")

    e := <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)

    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, unsupportedStompCommandError.Error()), true)

    assert.Equal(t, stompConn.state, closed)
    assert.Equal(t, rawConn.connected, false)
}

func TestStompConn_ConnectNoServerHeartbeat(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
            frame.CONNECT,
            frame.AcceptVersion, "1.1,1.0",
            frame.HeartBeat, "4000,4000")

    <- events

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED,
        frame.Version, "1.1",
        frame.HeartBeat, "0,0"), false)
}

func TestStompConn_ConnectServerHeartbeat(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(9999999991, []string{}), nil)
    rawConn.incomingFrames <- frame.New(
        frame.CONNECT,
        frame.AcceptVersion, "1.1,1.0",
        frame.HeartBeat, "4000,4000")

    <- events

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED,
        frame.Version, "1.1",
        frame.HeartBeat, "999999999,999999999"), false)
}

func TestStompConn_ConnectClientHeartbeat(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(7000, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
        frame.CONNECT,
        frame.AcceptVersion, "1.2",
        frame.HeartBeat, "8000,9000")

    <- events

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED,
        frame.HeartBeat, "9000,8000"), false)
}

func TestStompConn_ConnectWhenConnected(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED), false)

    rawConn.SendConnectFrame()

    e = <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.ERROR,
        frame.Message, unexpectedStompCommandError.Error()), true)

    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_SubscribeNotConnected(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
        frame.SUBSCRIBE,
        frame.Destination, "/topic/test")

    e := <- events
    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, e.conn, stompConn)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, notConnectedStompError.Error()), true)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_SubscribeMissingIdHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    rawConn.incomingFrames <- frame.New(
        frame.SUBSCRIBE,
        frame.Destination, "/topic/test")

    e = <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.ERROR,
        frame.Message, invalidSubscriptionError.Error()), true)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_SubscribeMissingDestinationHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    rawConn.incomingFrames <- frame.New(
        frame.SUBSCRIBE,
        frame.Id, "sub-id")

    e = <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.ERROR,
        frame.Message, invalidFrameError.Error()), true)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_Subscribe(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED), false)

    rawConn.incomingFrames <- frame.New(
        frame.SUBSCRIBE,
        frame.Id, "sub-id",
        frame.Destination, "/topic/test")

    e = <- events
    assert.Equal(t, e.eventType, SubscribeToTopic)
    assert.Equal(t, e.conn, stompConn)
    assert.Equal(t, e.destination, "/topic/test")
    assert.Equal(t, e.sub.destination, "/topic/test")
    assert.Equal(t, e.sub.id, "sub-id")
    assert.Equal(t, e.frame.Command, frame.SUBSCRIBE)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    assert.Equal(t, stompConn.state, connected)

    assert.Equal(t, stompConn.subscriptions["sub-id"].destination, "/topic/test")

    // trigger send subscribe request with the same id
    rawConn.incomingFrames <- frame.New(
        frame.SUBSCRIBE,
        frame.Id, "sub-id",
        frame.Destination, "/topic/test")

    rawConn.incomingFrames <- frame.New(frame.SEND, frame.Destination, "/topic/dest")

    // verify that there will be no SubscribeToTopic con event for the
    // the second request.
    e = <- events
    assert.Equal(t, e.eventType, IncomingMessage)
}

func TestStompConn_SendNotConnected(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
        frame.SEND,
        frame.Destination, "/topic/test")

    e := <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, notConnectedStompError.Error()), true)
}

func TestStompConn_SendMissingDestinationHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    rawConn.incomingFrames <- frame.New(
        frame.SEND)

    e = <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.ERROR,
        frame.Message, invalidFrameError.Error()), true)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_Send(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    msgF := frame.New(frame.SEND, frame.Destination, "/topic/test")

    rawConn.incomingFrames <- msgF

    e = <- events
    assert.Equal(t, e.eventType, IncomingMessage)
    assert.Equal(t, e.frame, msgF)
    assert.Equal(t, e.frame.Command, frame.MESSAGE)

    rawConn.incomingFrames <- frame.New(frame.SEND,
            frame.Destination, "/topic/test", frame.Receipt, "receipt-id")

    e = <- events
    assert.Equal(t, e.eventType, IncomingMessage)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.RECEIPT,
        frame.ReceiptId, "receipt-id"), true)
}

func TestStompConn_UnsubscribeNotConnected(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
        frame.UNSUBSCRIBE,
        frame.Destination, "/topic/test")

    e := <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, notConnectedStompError.Error()), true)
}

func TestStompConn_UnsubscribeMissingIdHeader(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    rawConn.incomingFrames <- frame.New(
        frame.UNSUBSCRIBE)

    e = <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.ERROR,
        frame.Message, invalidSubscriptionError.Error()), true)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_Unsubscribe(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    rawConn.incomingFrames <- frame.New(
        frame.UNSUBSCRIBE,
        frame.Id, "invalid-sub-id",
        frame.Receipt, "receipt-id")

    rawConn.incomingFrames <- frame.New(
        frame.SUBSCRIBE,
        frame.Id, "sub-id",
        frame.Destination, "/topic/test")

    e = <- events
    assert.Equal(t, e.eventType, SubscribeToTopic)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1], frame.New(frame.RECEIPT,
        frame.ReceiptId, "receipt-id"), true)

    rawConn.incomingFrames <- frame.New(
        frame.UNSUBSCRIBE,
        frame.Id, "sub-id")

    e = <- events
    assert.Equal(t, e.eventType, UnsubscribeFromTopic)
    assert.Equal(t, e.conn, stompConn)
    assert.Equal(t, e.destination, "/topic/test")
    assert.Equal(t, e.sub.destination, "/topic/test")
    assert.Equal(t, e.sub.id, "sub-id")

    assert.Equal(t, len(stompConn.subscriptions), 0)
    assert.Equal(t, stompConn.state, connected)
}

func TestStompConn_DisconnectNotConnected(t *testing.T) {
    _, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
        frame.DISCONNECT)

    e := <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.ERROR,
        frame.Message, notConnectedStompError.Error()), true)
}

func TestStompConn_Disconnect(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <- events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED), false)

    rawConn.incomingFrames <- frame.New(
        frame.DISCONNECT)

    e = <- events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_DisconnectWithReceipt(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <-events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    assert.Equal(t, len(rawConn.sentFrames), 1)
    verifyFrame(t, rawConn.sentFrames[0], frame.New(frame.CONNECTED), false)

    rawConn.incomingFrames <- frame.New(
        frame.DISCONNECT,
        frame.Receipt, "test-receipt")

    e = <-events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 2)
    verifyFrame(t, rawConn.sentFrames[1],
        frame.New(frame.RECEIPT, frame.ReceiptId, "test-receipt"), true)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_Close(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    stompConn.Close()

    e := <-events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 0)
    assert.Equal(t, stompConn.state, closed)
    assert.Equal(t, rawConn.connected, false)
}

func TestStompConn_SendFrameToSubscription(t *testing.T) {
    stompConn, rawConn, _ := getTestStompConn(NewStompConfig(0, []string{}), nil)

    sub := &subscription{
        id: "sub-id",
        destination: "/topic/test",
    }

    f := frame.New(frame.MESSAGE, frame.Destination, "/topic/test")

    rawConn.writeWg = &sync.WaitGroup{}
    rawConn.writeWg.Add(1)

    stompConn.SendFrameToSubscription(f, sub)

    rawConn.writeWg.Wait()
    assert.Equal(t, len(rawConn.sentFrames), 1)

    assert.Equal(t, rawConn.sentFrames[0], f)
    assert.Equal(t, rawConn.sentFrames[0].Header.Get(frame.MessageId), "1")

    rawConn.writeWg.Add(1)
    stompConn.SendFrameToSubscription(f, sub)
    rawConn.writeWg.Wait()
    assert.Equal(t, len(rawConn.sentFrames), 2)
    assert.Equal(t, rawConn.sentFrames[1].Header.Get(frame.MessageId), "2")

    rawConn.writeWg.Add(50)
    for i := 0; i < 50; i++ {
        go stompConn.SendFrameToSubscription(f.Clone(), sub)
    }

    rawConn.writeWg.Wait()
    assert.Equal(t, len(rawConn.sentFrames), 52)
}

func TestStompConn_SendErrorFrameToSubscription(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    sub := &subscription{
        id: "sub-id",
        destination: "/topic/test",
    }

    f := frame.New(frame.ERROR, frame.Destination, "/topic/test")
    stompConn.SendFrameToSubscription(f, sub)

    e := <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, len(rawConn.sentFrames), 1)
}

func TestStompConn_ReadError(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.incomingFrames <- errors.New("read error")

    e := <-events
    assert.Equal(t, e.eventType, ConnectionClosed)

    assert.Equal(t, len(rawConn.sentFrames), 0)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_WriteErrorDuringConnect(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.nextWriteErr = errors.New("write error")
    rawConn.SendConnectFrame()

    e := <-events
    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_WriteErrorDuringSend(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(0, []string{}), nil)

    rawConn.SendConnectFrame()

    e := <-events
    assert.Equal(t, e.eventType, ConnectionEstablished)

    rawConn.nextWriteErr = errors.New("write error")
    rawConn.incomingFrames <- frame.New(
        frame.SEND,
        frame.Destination, "/topic",
        frame.Receipt, "receipt-id")

    e = <- events

    assert.Equal(t, e.eventType, ConnectionClosed)
    assert.Equal(t, stompConn.state, closed)
}

func TestStompConn_SetReadDeadline(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(20000, []string{}), nil)

    infiniteTimeout := time.Time{}

    assert.Equal(t, rawConn.getCurrentReadDeadline(), infiniteTimeout)

    rawConn.incomingFrames <- frame.New(
        frame.CONNECT,
        frame.AcceptVersion, "1.2",
        frame.HeartBeat, "200,200")

    <-events

    // verify timeout is set to 20 seconds
    assert.Equal(t, stompConn.readTimeoutMs, int64(20000))

    rawConn.incomingFrames <- nil
    rawConn.incomingFrames <- nil

    diff := rawConn.getCurrentReadDeadline().Sub(time.Now())

    // verify the read deadline for the connection is
    // between 15 and 21 seconds
    assert.Greater(t, diff.Seconds(), float64(15))
    assert.Greater(t, float64(21), diff.Seconds())
}

func TestStompConn_WriteHeartbeat(t *testing.T) {
    stompConn, rawConn, events := getTestStompConn(NewStompConfig(100, []string{}), nil)

    rawConn.incomingFrames <- frame.New(
        frame.CONNECT,
        frame.AcceptVersion, "1.2",
        frame.HeartBeat, "50,50")

    <-events

    rawConn.lock.Lock()
    rawConn.writeWg = new(sync.WaitGroup)
    rawConn.writeWg.Add(2)
    rawConn.lock.Unlock()

    rawConn.writeWg.Wait()

    // verify the last frame is heartbeat (nil)
    rawConn.lock.Lock()
    assert.Nil(t, rawConn.LastSentFrame())
    rawConn.lock.Unlock()

    rawConn.writeWg.Add(3)
    stompConn.SendFrameToSubscription(frame.New(frame.MESSAGE), &subscription{id: "sub-1"})
    rawConn.writeWg.Wait()

    // verify the last frame is heartbeat (nil)
    rawConn.lock.Lock()
    rawConn.writeWg = nil
    assert.Nil(t, rawConn.LastSentFrame())
    rawConn.lock.Unlock()
}

func verifyFrame(t *testing.T, actualFrame *frame.Frame, expectedFrame *frame.Frame, exactHeaderMatch bool) {
    assert.Equal(t, expectedFrame.Command, actualFrame.Command)
    if exactHeaderMatch {
        assert.Equal(t, expectedFrame.Header.Len(), actualFrame.Header.Len())
    }

    for i := 0; i < expectedFrame.Header.Len(); i++ {
        key, value := expectedFrame.Header.GetAt(i);
        assert.Equal(t, actualFrame.Header.Get(key), value)
    }
}

func printFrame(f *frame.Frame) {
    if f == nil {
        fmt.Println("HEARTBEAT FRAME")
    } else {
        fmt.Println("FRAME:", f.Command)
        for i := 0; i < f.Header.Len(); i++ {
            key, value := f.Header.GetAt(i)
            fmt.Println(key + ":", value)
        }
        fmt.Println("BODY:", string(f.Body))
    }
}

// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package stompserver

import (
    "errors"
    "github.com/go-stomp/stomp/frame"
    "github.com/stretchr/testify/assert"
    "net"
    "testing"
    "time"
)

type MockTcpConnListener struct {
    serverCon net.Conn
    acceptError error
    connected bool
}

func (l *MockTcpConnListener) Accept() (net.Conn, error) {
    return l.serverCon, l.acceptError
}

func (l *MockTcpConnListener) Close() error {
    l.connected = false
    return errors.New("close-error")
}

func (l *MockTcpConnListener) Addr() net.Addr {
    return nil
}

func TestTcpConnectionListener_NewListenerInvalidAddr(t *testing.T) {
    tcpListener, err := NewTcpConnectionListener("invalid-addr")
    assert.Nil(t, tcpListener)
    assert.NotNil(t, err)
}

func TestTcpConnectionListener_Accept(t *testing.T) {

    serverConn, clientConn := net.Pipe()

    mockL := &MockTcpConnListener{
        connected: true,
        serverCon: serverConn,
        acceptError: nil,
    }

    tcpConListener := &tcpConnectionListener{listener: mockL}

    rawCon, err := tcpConListener.Accept()
    assert.NotNil(t, rawCon)
    assert.Nil(t, err)

    go func() {
        wr := frame.NewWriter(clientConn)
        wr.Write(frame.New(frame.CONNECT, frame.AcceptVersion, "1.2"))
    }()

    f, readErr := rawCon.ReadFrame()

    assert.Nil(t, readErr)
    verifyFrame(t, f, frame.New(frame.CONNECT, frame.AcceptVersion, "1.2"), true)

    frameReader := frame.NewReader(clientConn)

    go rawCon.WriteFrame(frame.New(frame.CONNECTED, frame.Version, "1.2"))

    serverFrame, writeErr := frameReader.Read()
    assert.NotNil(t, serverFrame)
    assert.Nil(t, writeErr)

    verifyFrame(t, serverFrame, frame.New(frame.CONNECTED, frame.Version, "1.2"), true)

    rawCon.SetReadDeadline(time.Now().Add(time.Duration(-1) * time.Second))
    _, timeoutErr := rawCon.ReadFrame()

    assert.NotNil(t, timeoutErr)

    rawCon.SetReadDeadline(time.Time{})

    assert.Nil(t, rawCon.Close())

    n, _ := serverConn.Read(make([]byte, 1))
    assert.Equal(t, n, 0)

    assert.EqualError(t, tcpConListener.Close(), "close-error")
}

func TestTcpConnectionListener_AcceptError(t *testing.T) {
    mockL := &MockTcpConnListener{
        connected: true,
        serverCon: nil,
        acceptError: errors.New("accept-error"),
    }

    tcpConListener := &tcpConnectionListener{listener: mockL}

    rawCon, err := tcpConListener.Accept()
    assert.Nil(t, rawCon)
    assert.EqualError(t, err, "accept-error")
}


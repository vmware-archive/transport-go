// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package stompserver

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/gorilla/websocket"
    "github.com/go-stomp/stomp/frame"
    "time"
    "sync"
    "net/http"
)

func TestWebSocketConnectionListener_NewListenerInvalidAddr(t *testing.T) {
    wsListener, err := NewWebSocketConnectionListener("invalid-addr", "/fabric", nil)
    assert.Nil(t, wsListener)
    assert.NotNil(t, err)
}

func TestWebSocketConnectionListener_CheckOrigin(t *testing.T) {
    wsListener := &webSocketConnectionListener{
        allowedOrigins: nil,
    }

    r := new(http.Request)
    r.Header = make(http.Header)
    r.Header["Origin"] = []string {"http://localhost:4200"}
    r.Host = "localhost:8000"

    assert.Equal(t, wsListener.checkOrigin(r), true)

    wsListener.allowedOrigins = make([]string, 0)
    assert.Equal(t, wsListener.checkOrigin(r), true)

    wsListener.allowedOrigins = []string {"appfabric.eng.vmware.com:4200"}
    assert.Equal(t, wsListener.checkOrigin(r), false)

    wsListener.allowedOrigins = []string {"appfabric.eng.vmware.com:4200", "localhost:4200"}
    assert.Equal(t, wsListener.checkOrigin(r), true)

    wsListener.allowedOrigins = []string {"appfabric.eng.vmware.com:4200"}
    r.Host = "localhost:4200"
    assert.Equal(t, wsListener.checkOrigin(r), true)

    r.Header["Origin"] = []string {}
    assert.Equal(t, wsListener.checkOrigin(r), true)

    r.Header["Origin"] = []string {"http://192.168.0.%31/"}
    assert.Equal(t, wsListener.checkOrigin(r), false)
}

func TestWebSocketConnectionListener_NewListener(t *testing.T) {

    listener, err := NewWebSocketConnectionListener("", "/fabric", []string{"localhost:8000"})

    assert.Nil(t, err)
    assert.NotNil(t, listener)

    wsListener := listener.(*webSocketConnectionListener)

    wg := sync.WaitGroup{}
    wg.Add(1)

    dialer := &websocket.Dialer{}
    var clientConn *websocket.Conn
    go func() {
        var err error
        clientConn, _, err = dialer.Dial(
            "ws://" + wsListener.tcpConnectionListener.Addr().String() + "/fabric", nil)
        assert.NotNil(t, clientConn)
        assert.Nil(t, err)

        wg.Done()
    }()

    rawConn, err := listener.Accept()
    assert.NotNil(t, rawConn)
    assert.Nil(t, err)

    wg.Wait()

    go func() {
        wsWriter, _ := clientConn.NextWriter(websocket.TextMessage)
        wr := frame.NewWriter(wsWriter)
        wr.Write(frame.New(frame.CONNECT, frame.AcceptVersion, "1.2"))
        wsWriter.Close()
    }()

    f, e := rawConn.ReadFrame()

    assert.NotNil(t, f)
    assert.Nil(t, e)

    verifyFrame(t, f, frame.New(frame.CONNECT, frame.AcceptVersion, "1.2"), true)

    wg.Add(1)

    go func() {
       rawConn.WriteFrame(frame.New(frame.CONNECTED, frame.Version, "1.2"))
       wg.Done()
    }()

    _, wsReader, _ := clientConn.NextReader()
    serverFrame, err := frame.NewReader(wsReader).Read()
    assert.NotNil(t, serverFrame)
    assert.Nil(t, err)

    wg.Wait()

    verifyFrame(t, serverFrame, frame.New(frame.CONNECTED, frame.Version, "1.2"), true)

    rawConn.SetReadDeadline(time.Now().Add(time.Duration(-1) * time.Second))
    _, timeoutErr := rawConn.ReadFrame()

    assert.NotNil(t, timeoutErr)

    rawConn.SetReadDeadline(time.Time{})

    assert.Nil(t, rawConn.Close())

    _, reader, err := clientConn.NextReader()
    assert.Nil(t, reader)
    assert.NotNil(t, err)

    err = rawConn.WriteFrame(frame.New(frame.MESSAGE))
    assert.NotNil(t, err)

    var failedClientConn *websocket.Conn
    wg.Add(1)
    go func() {
        requestHeaders := make(http.Header)
        requestHeaders["Origin"] = []string{"http://192.168.0.%31/"}

        var err error
        failedClientConn, _, err = dialer.Dial(
            "ws://" + wsListener.tcpConnectionListener.Addr().String() + "/fabric", requestHeaders)
        assert.Nil(t, failedClientConn)
        assert.NotNil(t, err)
        wg.Done()
    }()

    failedConn, connErr := listener.Accept()
    assert.Nil(t, failedConn)
    assert.NotNil(t, connErr)

    wg.Wait()

    listener.Close()

    clientConn2, _, err := dialer.Dial(
        "ws://" + wsListener.tcpConnectionListener.Addr().String() + "/fabric", nil)
    assert.Nil(t, clientConn2)
    assert.NotNil(t, err)
}

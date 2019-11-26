// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package stompserver

import (
    "errors"
    "fmt"
    "github.com/go-stomp/stomp/frame"
    "github.com/stretchr/testify/assert"
    "strconv"
    "sync"
    "testing"
)

type MockRawConnectionListener struct {
    connected bool
    incomingConnections chan interface{}
}

func NewMockRawConnectionListener() *MockRawConnectionListener {
    return &MockRawConnectionListener{
       incomingConnections: make(chan interface{}),
       connected: true,
    }
}

func (cl *MockRawConnectionListener) Accept() (RawConnection, error) {
    obj := <- cl.incomingConnections
    mockConn, ok := obj.(*MockRawConnection)
    if ok {
        return mockConn, nil
    }

    return nil, obj.(error)
}

func (cl *MockRawConnectionListener) Close() error {
    cl.connected = false
    return nil
}

func newTestStompServer(config StompConfig) (*stompServer, *MockRawConnectionListener) {
    listener := NewMockRawConnectionListener()
    return NewStompServer(listener, config).(*stompServer), listener

}

func TestStompServer_NewSubscription(t *testing.T) {
    server, conListener := newTestStompServer(NewStompConfig(0, []string {"/pub"}))

    go server.Start()

    wg := sync.WaitGroup{}
    wg.Add(1)

    server.OnSubscribeEvent(
        func(conId string, subId string, destination string, frame *frame.Frame) {
            assert.Equal(t, subId, "sub-id-1")
            wg.Done()
        })

    mockRawConn := NewMockRawConnection()
    conListener.incomingConnections <- mockRawConn

    mockRawConn.SendConnectFrame()
    mockRawConn.incomingFrames <- frame.New(frame.SUBSCRIBE,
        frame.Destination, "/topic/destination",
        frame.Id, "sub-id-1")

    wg.Wait()
}

func TestStompServer_OnApplicationRequest(t *testing.T) {
    appPrefixes := []string{"/pub", "/pub2", "/pub3/"}
    server, _ := newTestStompServer(NewStompConfig(0, appPrefixes))

    assert.Equal(t, server.config.AppDestinationPrefix(), []string{"/pub/", "/pub2/", "/pub3/"})
    assert.Equal(t, appPrefixes, []string{"/pub", "/pub2", "/pub3/"})
    fmt.Println(appPrefixes)

    go server.Start()

    wg := sync.WaitGroup{}

    wg.Add(2)
    server.OnApplicationRequest(func(destination string, message []byte, connectionId string) {
        if destination == "/pub/testRequest1" {
            assert.Equal(t, string(message), "request1-payload")
            assert.Equal(t, connectionId, "con1")
            wg.Done()
        } else if destination == "/pub2/testRequest2" {
            assert.Equal(t, string(message), "request2-payload")
            assert.Equal(t, connectionId, "con2")
            wg.Done()
        } else {
            assert.Fail(t, "unexpected request")
        }
    })

    f1 := frame.New(frame.MESSAGE, frame.Destination, "/pub/testRequest1")
    f1.Body = []byte("request1-payload")

    f2 := frame.New(frame.MESSAGE, frame.Destination, "/pub2/testRequest2")
    f2.Body = []byte("request2-payload")

    server.connectionEvents <- &connEvent{
        eventType: incomingMessage,
        conn: &stompConn{
            id: "con1",
        },
        destination: "/pub/testRequest1",
        frame: f1,
    }
    server.connectionEvents <- &connEvent{
        eventType: incomingMessage,
        conn: &stompConn{
            id: "con2",
        },
        destination: "/pub2/testRequest2",
        frame: f2,
    }
    server.connectionEvents <- &connEvent{
        eventType: incomingMessage,
        conn: &stompConn{
            id: "con1",
        },
        destination: "/pub4/testRequest3",
        frame: frame.New(frame.MESSAGE, frame.Destination, "/pub3/testRequest3"),
    }

    wg.Wait()
}

func TestStompServer_SendMessage(t *testing.T) {
    server, listener := newTestStompServer(NewStompConfig(0, []string{"/pub/"}))
    go server.Start()

    mockRwConn1 := NewMockRawConnection()
    mockRwConn2 := NewMockRawConnection()
    mockRwConn3 := NewMockRawConnection()

    listener.incomingConnections <- mockRwConn1
    listener.incomingConnections <- mockRwConn2
    listener.incomingConnections <- mockRwConn3

    mockRwConn1.SendConnectFrame()
    mockRwConn2.SendConnectFrame()
    mockRwConn3.SendConnectFrame()

    wg := sync.WaitGroup{}
    wg.Add(6)
    server.OnSubscribeEvent(func(conId string, subId string, destination string, f *frame.Frame) {
        wg.Done()
    })

    subscribeMockConToTopic(mockRwConn1, "/topic/test-topic1", "/topic/test-topic1", "/topic/test-topic2")
    subscribeMockConToTopic(mockRwConn2, "/topic/test-topic1", "/topic/test-topic3")
    subscribeMockConToTopic(mockRwConn3, "/topic/test-topic3")

    wg.Wait()

    mockRwConn1.writeWg = &wg
    mockRwConn2.writeWg = &wg
    mockRwConn3.writeWg = &wg

    wg.Add(3)
    server.SendMessage("/topic/test-topic1", []byte("test-message"))
    wg.Wait()

    f := mockRwConn1.LastSentFrame()
    assert.Equal(t, len(mockRwConn1.sentFrames), 3)
    verifyFrame(t, f, frame.New(
        frame.MESSAGE, frame.Destination, "/topic/test-topic1"), false)
    assert.Equal(t, string(f.Body), "test-message")

    f = mockRwConn2.LastSentFrame()
    assert.Equal(t, len(mockRwConn2.sentFrames), 2)
    verifyFrame(t, f, frame.New(frame.MESSAGE,
            frame.Destination, "/topic/test-topic1",
            frame.Subscription, "/topic/test-topic1-0"), false)
    assert.Equal(t, string(f.Body), "test-message")

    assert.Equal(t, len(mockRwConn3.sentFrames), 1)

    wg.Add(2)
    server.SendMessage("/topic/test-topic3", []byte("test-message2"))
    wg.Wait()

    assert.Equal(t, len(mockRwConn1.sentFrames), 3)

    assert.Equal(t, len(mockRwConn2.sentFrames), 3)
    assert.Equal(t, string(mockRwConn2.LastSentFrame().Body), "test-message2")

    assert.Equal(t, len(mockRwConn3.sentFrames), 2)
    assert.Equal(t, string(mockRwConn3.LastSentFrame().Body), "test-message2")

    wg.Add(1)
    server.SendMessage("/topic/test-topic2", []byte("test-message3"))
    wg.Wait()

    assert.Equal(t, len(mockRwConn1.sentFrames), 4)
    assert.Equal(t, string(mockRwConn1.LastSentFrame().Body), "test-message3")
    assert.Equal(t, len(mockRwConn2.sentFrames), 3)
    assert.Equal(t, len(mockRwConn3.sentFrames), 2)


    server.OnUnsubscribeEvent(func(conId string, subId string, destination string) {
        wg.Done()
    })

    wg.Add(1)
    mockRwConn1.incomingFrames <- frame.New(
        frame.UNSUBSCRIBE, frame.Id, "/topic/test-topic1-0")
    wg.Wait()


    wg.Add(2)
    server.SendMessage("/topic/test-topic1", []byte("test-message4"))
    wg.Wait()

    assert.Equal(t, len(mockRwConn1.sentFrames), 5)
    assert.Equal(t, string(mockRwConn1.LastSentFrame().Body), "test-message4")
    assert.Equal(t, len(mockRwConn2.sentFrames), 4)
    assert.Equal(t, string(mockRwConn2.LastSentFrame().Body), "test-message4")

    wg.Add(1)
    mockRwConn1.incomingFrames <- frame.New(
        frame.UNSUBSCRIBE, frame.Id, "/topic/test-topic1-1")
    wg.Wait()

    wg.Add(1)
    server.SendMessage("/topic/test-topic1", []byte("test-message5"))
    wg.Wait()

    assert.Equal(t, len(mockRwConn1.sentFrames), 5)
    assert.Equal(t, len(mockRwConn2.sentFrames), 5)
    assert.Equal(t, string(mockRwConn2.LastSentFrame().Body), "test-message5")
    assert.Equal(t, len(mockRwConn3.sentFrames), 2)

    wg.Add(2)
    mockRwConn2.incomingFrames <- frame.New(frame.DISCONNECT)
    wg.Wait()

    wg.Add(1)
    server.SendMessage("/topic/test-topic3", []byte("test-message6"))
    wg.Wait()

    assert.Equal(t, len(mockRwConn1.sentFrames), 5)
    assert.Equal(t, len(mockRwConn2.sentFrames), 5)
    assert.Equal(t, len(mockRwConn3.sentFrames), 3)
    assert.Equal(t, string(mockRwConn3.LastSentFrame().Body), "test-message6")

    wg.Add(2)
    server.OnUnsubscribeEvent(func(conId string, subId string, destination string) {
        assert.Equal(t, subId, "/topic/test-topic3-0")
        assert.Equal(t, destination, "/topic/test-topic3")
        wg.Done()
    })
    mockRwConn3.incomingFrames <- frame.New(frame.DISCONNECT)
    wg.Wait()
}

func TestStompServer_SendMessageToClient(t *testing.T) {
    server, listener := newTestStompServer(NewStompConfig(0, []string{"/pub/"}))
    go server.Start()

    mockRwConn1 := NewMockRawConnection()
    mockRwConn2 := NewMockRawConnection()

    listener.incomingConnections <- mockRwConn1
    listener.incomingConnections <- mockRwConn2

    mockRwConn1.SendConnectFrame()
    mockRwConn2.SendConnectFrame()

    wg := sync.WaitGroup{}
    wg.Add(5)
    server.OnSubscribeEvent(func(conId string, subId string, destination string, f *frame.Frame) {
        wg.Done()
    })

    subscribeMockConToTopic(mockRwConn1, "/topic/test-topic1", "/topic/test-topic1", "/topic/test-topic2")
    subscribeMockConToTopic(mockRwConn2, "/topic/test-topic1", "/topic/test-topic3")

    wg.Wait()

    connMap := make(map[*MockRawConnection]*stompConn)
    for _, conn := range server.connectionsMap {
        if conn.(*stompConn).rawConnection == mockRwConn1 {
            connMap[mockRwConn1] = conn.(*stompConn)
        } else if conn.(*stompConn).rawConnection == mockRwConn2 {
            connMap[mockRwConn2] = conn.(*stompConn)
        }
    }

    mockRwConn1.writeWg = &wg
    mockRwConn2.writeWg = &wg

    wg.Add(2)
    server.SendMessageToClient(connMap[mockRwConn1].id, "/topic/test-topic1", []byte("test-message"))
    wg.Wait()

    f := mockRwConn1.LastSentFrame()
    assert.Equal(t, len(mockRwConn1.sentFrames), 3)
    verifyFrame(t, f, frame.New(
        frame.MESSAGE, frame.Destination, "/topic/test-topic1"), false)
    assert.Equal(t, string(f.Body), "test-message")

    assert.Equal(t, len(mockRwConn2.sentFrames), 1)

    server.SendMessageToClient(connMap[mockRwConn2].id, "/topic/invalid-topic", []byte("invalid-message"))

    server.SendMessageToClient("invalid-connection-id", "/topic/invalid-topic", []byte("invalid-message"))

    wg.Add(1)
    server.SendMessageToClient(connMap[mockRwConn2].id, "/topic/test-topic1", []byte("test-message2"))
    wg.Wait()

    assert.Equal(t, len(mockRwConn1.sentFrames), 3)
    assert.Equal(t, len(mockRwConn2.sentFrames), 2)
    assert.Equal(t, string(mockRwConn2.LastSentFrame().Body), "test-message2")


}

func TestStompServer_Stop(t *testing.T) {
    server, listener := newTestStompServer(NewStompConfig(0, []string{"/pub"}))

    wg := sync.WaitGroup{}

    go func() {
        server.Start()
        wg.Done()
    }()

    mockRwConn1 := NewMockRawConnection()
    mockRwConn2 := NewMockRawConnection()
    listener.incomingConnections <- mockRwConn1
    listener.incomingConnections <- mockRwConn2

    wg.Add(2)
    server.OnSubscribeEvent(func(conId string, subId string, destination string, frame *frame.Frame) {
        wg.Done()
    })

    mockRwConn1.SendConnectFrame()
    mockRwConn2.SendConnectFrame()
    subscribeMockConToTopic(mockRwConn1, "/topic1")
    subscribeMockConToTopic(mockRwConn1, "/topic2")

    wg.Wait()

    // calling start on started server doesn't do anything and doesn't block
    server.Start()

    wg.Add(1)
    server.Stop()
    wg.Wait()

    assert.Equal(t, mockRwConn1.connected, false)
    assert.Equal(t, mockRwConn2.connected, false)
    assert.Equal(t, listener.connected, false)
}

func TestStompServer_ConnectionErrors(t *testing.T) {
    server, listener := newTestStompServer(NewStompConfig(0, []string{"/pub"}))

    go server.Start()

    // simulate 10 errors
    for i := 0; i < 10; i++ {
        listener.incomingConnections <- errors.New("connection-error")
    }

    assert.Equal(t, len(server.connectionsMap), 0)

    // verify that we can still connect after the errors
    for i := 0; i < 5; i++ {
        listener.incomingConnections <- NewMockRawConnection()
    }

    server.callbackLock.Lock()
    defer server.callbackLock.Unlock()
    assert.True(t, len(server.connectionsMap) > 0)
}

func subscribeMockConToTopic(conn *MockRawConnection, topics ...string) {
    for index, topic := range topics {
        conn.incomingFrames <- frame.New(frame.SUBSCRIBE,
            frame.Destination, topic,
            frame.Id, topic + "-" + strconv.Itoa(index))
    }
}

// Copyright 2019 VMware Inc.
package bus

import (
    "go-bifrost/bridge"
    "go-bifrost/model"
    "go-bifrost/util"
    "bufio"
    "bytes"
    "errors"
    "fmt"
    "github.com/go-stomp/stomp/frame"
    "github.com/go-stomp/stomp/server"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    "log"
    "net"
    "net/http"
    "net/http/httptest"
    "net/url"
    "testing"
    "sync/atomic"
)

var evtBusTest *bifrostEventBus
var evtbusTestChannelName string = "test-channel"
var evtbusTestManager ChannelManager

var testBrokerAddress = "localhost:56966"
var httpServer *httptest.Server
var webSocketURLChan = make(chan string, 1)
var websocketURL string
var upgrader = websocket.Upgrader{}
var tcpReady = make(chan bool, 1)

func runWebSocketEndPoint() string {
    s := httptest.NewServer(http.HandlerFunc(websocketHandler))
    log.Println("WebSocket listening on", s.Listener.Addr().Network(), s.Listener.Addr().String())
    httpServer = s
    websocketURL = s.URL
    return s.URL
}

var tcpServer net.Listener

func runStompBroker() {
    l, err := net.Listen("tcp", testBrokerAddress)
    if err != nil {
        log.Fatalf("failed to listen: %s", err.Error())
    }
    defer func() { l.Close() }()

    log.Println("TCP listening on", l.Addr().Network(), l.Addr().String())
    server.Serve(l)
    tcpServer = l
}

func killWebSocketEndpoint() {
    httpServer.Close()
}

// upgrade http connection to WS and read/write responses.
func websocketHandler(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer c.Close()
    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            break
        }

        br := bytes.NewReader(message)
        sr := frame.NewReader(br)
        f, _ := sr.Read()

        var sendFrame *frame.Frame

        switch f.Command {
        case frame.CONNECT:
            sendFrame = frame.New(frame.CONNECTED,
                frame.ContentType, "application/json")

        case frame.SUBSCRIBE:
            sendFrame = frame.New(frame.MESSAGE,
                frame.Destination, f.Header.Get(frame.Destination),
                frame.ContentType, "application/json")
            sendFrame.Body = []byte("happy baby melody!")

        case frame.UNSUBSCRIBE:
            sendFrame = frame.New(frame.MESSAGE,
                frame.Destination, f.Header.Get(frame.Destination),
                frame.ContentType, "application/json")
            sendFrame.Body = []byte("bye bye!")
        }
        var bb bytes.Buffer
        bw := bufio.NewWriter(&bb)
        sw := frame.NewWriter(bw)
        sw.Write(sendFrame)

        err = c.WriteMessage(mt, bb.Bytes())
        if err != nil {
            break
        }
    }
}


func init() {
    evtBusTest = GetBus().(*bifrostEventBus)

}

func createTestChannel() *Channel {

    //create new bus
    //bf := new(bifrostEventBus)
    //bf.init()
    //evtBusTest = bf
    //busInstance = bf // set GetBus() instance to return new instance also.

    evtbusTestManager = evtBusTest.GetChannelManager()
    return evtbusTestManager.CreateChannel(evtbusTestChannelName)
}

func inc(counter *int32) {
    atomic.AddInt32(counter, 1)
}

func destroyTestChannel() {
    evtbusTestManager.DestroyChannel(evtbusTestChannelName)
    util.ResetMonitor()
}

func TestEventBus_Boot(t *testing.T) {

    bus1 := GetBus()
    bus2 := GetBus()
    bus3 := GetBus()

    assert.EqualValues(t, bus1.GetId(), bus2.GetId())
    assert.EqualValues(t, bus2.GetId(), bus3.GetId())
    assert.NotNil(t, evtBusTest.GetChannelManager())
}

func TestEventBus_SendResponseMessageNoChannel(t *testing.T) {
    err := evtBusTest.SendResponseMessage("Channel-not-here", "hello melody", nil)
    assert.NotNil(t, err)
}

func TestEventBus_SendRequestMessageNoChannel(t *testing.T) {
    err := evtBusTest.SendRequestMessage("Channel-not-here", "hello melody", nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenStream(t *testing.T) {
    createTestChannel()
    handler, err := evtBusTest.ListenStream(evtbusTestChannelName)
    assert.Nil(t, err)
    assert.NotNil(t, handler)
    var count int32 = 0
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            inc(&count)
        },
        func(err error) {})

    for i := 0; i < 3; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "hello melody", nil)

        // send requests to make sure we're only getting requests
        //evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, nil)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1, nil)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, int32(3), count)
    destroyTestChannel()
}

func TestBifrostEventBus_ListenStreamForDestination(t *testing.T) {
    createTestChannel()
    id := uuid.New()
    handler, _ := evtBusTest.ListenStreamForDestination(evtbusTestChannelName, &id)
    var count int32 = 0
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            inc(&count)
        },
        func(err error) {})

    for i := 0; i < 20; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "hello melody", &id)

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, &id)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1, &id)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, int32(20), count)
    destroyTestChannel()
}

func TestEventBus_ListenStreamNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenStream("missing-Channel")
    assert.NotNil(t, err)
}

func TestEventBus_ListenOnce(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenOnce(evtbusTestChannelName)
    count := 0
    handler.Handle(
        func(msg *model.Message) {
            count++
        },
        func(err error) {})

    for i := 0; i < 10; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, handler.GetDestinationId())
    }

    for i := 0; i < 2; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 0, handler.GetDestinationId())

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, handler.GetDestinationId())
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1, handler.GetDestinationId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_ListenOnceForDestination(t *testing.T) {
    createTestChannel()
    dest := uuid.New()
    handler, _ := evtBusTest.ListenOnceForDestination(evtbusTestChannelName, &dest)
    count := 0
    handler.Handle(
        func(msg *model.Message) {
            count++
        },
        func(err error) {})

    for i := 0; i < 300; i++ {
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 0, &dest)

        // send duplicate
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 0, &dest)

        // send random noise
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 0, nil)

        // send requests to make sure we're only getting requests
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, &dest)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 1, &dest)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_ListenOnceNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenOnce("missing-Channel")
    assert.NotNil(t, err)
}

func TestEventBus_ListenOnceForDestinationNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenOnceForDestination("missing-Channel", nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenOnceForDestinationNoDestination(t *testing.T) {
    createTestChannel()
    _, err := evtBusTest.ListenOnceForDestination(evtbusTestChannelName, nil)
    assert.NotNil(t, err)
    destroyTestChannel()
}

func TestEventBus_ListenRequestStream(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenRequestStream(evtbusTestChannelName)
    var count int32 = 0
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            inc(&count)
        },
        func(err error) {})

    for i := 0; i < 10000; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody", nil)

        // send responses to make sure we're only getting requests
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion if picked up", nil)
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion again", nil)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, count, int32(10000))
    destroyTestChannel()
}

func TestEventBus_ListenRequestStreamForDestination(t *testing.T) {
    createTestChannel()
    id := uuid.New()
    handler, _ := evtBusTest.ListenRequestStreamForDestination(evtbusTestChannelName, &id)
    var count int32 = 0
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            inc(&count)
        },
        func(err error) {})

    for i := 0; i < 1000; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody", &id)

        // send responses to make sure we're only getting requests
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion if picked up", &id)
        evtBusTest.SendResponseMessage(evtbusTestChannelName, "will fail assertion again", &id)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, count, int32(1000))
    destroyTestChannel()
}

func TestEventBus_ListenStreamForDestinationNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenStreamForDestination("missing-Channel", nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenStreamForDestinationNoDestination(t *testing.T) {
    createTestChannel()
    _, err := evtBusTest.ListenStreamForDestination(evtbusTestChannelName, nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestStreamForDestinationNoDestination(t *testing.T) {
    createTestChannel()
    _, err := evtBusTest.ListenRequestStreamForDestination(evtbusTestChannelName, nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestStreamForDestinationNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenRequestStreamForDestination("nowhere", nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestOnce(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenRequestOnce(evtbusTestChannelName)
    count := 0
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 5; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody", handler.GetDestinationId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_ListenRequestOnceForDestination(t *testing.T) {
    createTestChannel()
    dest := uuid.New()
    handler, _ := evtBusTest.ListenRequestOnceForDestination(evtbusTestChannelName, &dest)
    count := 0
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "hello melody", msg.Payload.(string))
            count++
        },
        func(err error) {})

    for i := 0; i < 5; i++ {
        evtBusTest.SendRequestMessage(evtbusTestChannelName, "hello melody", &dest)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_ListenRequestOnceNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenRequestOnce("missing-Channel")
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestStreamNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenRequestStream("missing-Channel")
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestOnceForDestinationNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenRequestOnceForDestination("missing-Channel", nil)
    assert.NotNil(t, err)
}

func TestEventBus_ListenRequestOnceForDestinationNoDestination(t *testing.T) {
    createTestChannel()
    _, err := evtBusTest.ListenRequestOnceForDestination(evtbusTestChannelName, nil)
    assert.NotNil(t, err)
    destroyTestChannel()
}

func TestEventBus_TestErrorMessageHandling(t *testing.T) {
    createTestChannel()

    err := evtBusTest.SendErrorMessage("invalid-Channel", errors.New("something went wrong"), nil)
    assert.NotNil(t, err)

    handler, _ := evtBusTest.ListenStream(evtbusTestChannelName)
    var countError int32 = 0
    handler.Handle(
        func(msg *model.Message) {},
        func(err error) {
            assert.Errorf(t, err, "something went wrong")
            inc(&countError)
        })

    for i := 0; i < 5; i++ {
        err := errors.New("something went wrong")
        evtBusTest.SendErrorMessage(evtbusTestChannelName, err, handler.GetId())
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, int32(5), countError)
    destroyTestChannel()
}

func TestEventBus_ListenFirehose(t *testing.T) {
    createTestChannel()
    var counter int32 = 0

    responseHandler, _ := evtBusTest.ListenFirehose(evtbusTestChannelName)
    responseHandler.Handle(
        func(msg *model.Message) {
            inc(&counter)
        },
        func(err error) {
            inc(&counter)
        })
    for i := 0; i < 5; i++ {
        err := errors.New("something went wrong")
        evtBusTest.SendErrorMessage(evtbusTestChannelName, err, nil)
        evtBusTest.SendRequestMessage(evtbusTestChannelName, 0, nil)
        evtBusTest.SendResponseMessage(evtbusTestChannelName, 1, nil)
    }
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, counter, int32(15))
    destroyTestChannel()
}

func TestEventBus_ListenFirehoseNoChannel(t *testing.T) {
    _, err := evtBusTest.ListenFirehose("missing-Channel")
    assert.NotNil(t, err)
}

func TestEventBus_RequestOnce(t *testing.T) {
    createTestChannel()
    handler, _ := evtBusTest.ListenRequestStream(evtbusTestChannelName)
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "who is a pretty baby?", msg.Payload.(string))
            evtBusTest.SendResponseMessage(evtbusTestChannelName, "why melody is of course", msg.DestinationId)
        },
        func(err error) {})

    count := 0
    responseHandler, _ := evtBusTest.RequestOnce(evtbusTestChannelName, "who is a pretty baby?")
    responseHandler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "why melody is of course", msg.Payload.(string))
            count++
        },
        func(err error) {})

    responseHandler.Fire()
    evtbusTestManager.WaitForChannel(evtbusTestChannelName)
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_RequestOnceForDestination(t *testing.T) {
    createTestChannel()
    dest := uuid.New()
    handler, _ := evtBusTest.ListenRequestStream(evtbusTestChannelName)
    handler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "who is a pretty baby?", msg.Payload.(string))
            evtBusTest.SendResponseMessage(evtbusTestChannelName, "why melody is of course", msg.DestinationId)
        },
        func(err error) {})

    count := 0
    responseHandler, _ := evtBusTest.RequestOnceForDestination(evtbusTestChannelName, "who is a pretty baby?", &dest)
    responseHandler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "why melody is of course", msg.Payload.(string))
            count++
        },
        func(err error) {})

    responseHandler.Fire()
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_RequestOnceForDesintationNoChannel(t *testing.T) {
    _, err := evtBusTest.RequestOnceForDestination("some-chan", nil, nil)
    assert.NotNil(t, err)
}

func TestEventBus_RequestOnceForDesintationNoDestination(t *testing.T) {
    createTestChannel()
    _, err := evtBusTest.RequestOnceForDestination(evtbusTestChannelName, nil, nil)
    assert.NotNil(t, err)
    destroyTestChannel()
}

func TestEventBus_RequestStream(t *testing.T) {
    channel := createTestChannel()
    handler := func(message *model.Message) {
        if message.Direction == model.RequestDir {
            assert.Equal(t, "who has the cutest laugh?", message.Payload.(string))
            config := buildConfig(channel.Name, "why melody does of course", message.DestinationId)

            // fire a few times, ensure that the handler only ever picks up a single response.
            for i := 0; i < 5; i++ {
                channel.Send(model.GenerateResponse(config))
            }
        }
    }
    id := uuid.New()
    channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

    var count int32 = 0
    responseHandler, _ := evtBusTest.RequestStream(evtbusTestChannelName, "who has the cutest laugh?")
    responseHandler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "why melody does of course", msg.Payload.(string))
            inc(&count)
        },
        func(err error) {})

    responseHandler.Fire()
    assert.Equal(t, int32(5), count)
    destroyTestChannel()
}

func TestEventBus_RequestStreamForDesintation(t *testing.T) {
    channel := createTestChannel()
    dest := uuid.New()
    handler := func(message *model.Message) {
        if message.Direction == model.RequestDir {
            assert.Equal(t, "who has the cutest laugh?", message.Payload.(string))
            config := buildConfig(channel.Name, "why melody does of course", message.DestinationId)

            // fire a few times, ensure that the handler only ever picks up a single response.
            for i := 0; i < 5; i++ {
                channel.Send(model.GenerateResponse(config))
            }
        }
    }
    id := uuid.New()
    channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: false, uuid: &id})

    var count int32 = 0
    responseHandler, _ := evtBusTest.RequestStreamForDestination(evtbusTestChannelName, "who has the cutest laugh?", &dest)
    responseHandler.Handle(
        func(msg *model.Message) {
            assert.Equal(t, "why melody does of course", msg.Payload.(string))
            inc(&count)
        },
        func(err error) {})

    responseHandler.Fire()
    assert.Equal(t, int32(5), count)
    destroyTestChannel()
}

func TestEventBus_RequestStreamForDestinationNoChannel(t *testing.T) {
    _, err := evtBusTest.RequestStreamForDestination("missing-Channel", nil, nil)
    assert.NotNil(t, err)
}

func TestEventBus_RequestStreamForDestinationNoDestination(t *testing.T) {
    createTestChannel()
    _, err := evtBusTest.RequestStreamForDestination(evtbusTestChannelName, nil, nil)
    assert.NotNil(t, err)
    destroyTestChannel()
}

func TestEventBus_RequestStreamNoChannel(t *testing.T) {
    _, err := evtBusTest.RequestStream("missing-Channel", nil)
    assert.NotNil(t, err)
}

func TestEventBus_HandleSingleRunError(t *testing.T) {
    channel := createTestChannel()
    handler := func(message *model.Message) {
        if message.Direction == model.RequestDir {
            config := buildError(channel.Name, fmt.Errorf("whoops!"), message.DestinationId)

            // fire a few times, ensure that the handler only ever picks up a single response.
            for i := 0; i < 5; i++ {
                channel.Send(model.GenerateError(config))
            }
        }
    }
    id := uuid.New()
    channel.subscribeHandler(&channelEventHandler{callBackFunction: handler, runOnce: true, uuid: &id})

    count := 0
    responseHandler, _ := evtBusTest.RequestOnce(evtbusTestChannelName, 0)
    responseHandler.Handle(
        func(msg *model.Message) {},
        func(err error) {
            assert.Error(t, err, "whoops!")
            count++
        })

    responseHandler.Fire()
    assert.Equal(t, 1, count)
    destroyTestChannel()
}

func TestEventBus_RequestOnceNoChannel(t *testing.T) {
    _, err := evtBusTest.RequestOnce("missing-Channel", 0)
    assert.NotNil(t, err)
}

func TestEventBus_HandlerWithoutRequestToFire(t *testing.T) {
    createTestChannel()
    responseHandler, _ := evtBusTest.ListenFirehose(evtbusTestChannelName)
    responseHandler.Handle(
        func(msg *model.Message) {},
        func(err error) {})
    err := responseHandler.Fire()
    assert.Errorf(t, err, "nothing to fire, request is empty")
    destroyTestChannel()
}

func TestEventBus_GetStoreManager(t *testing.T) {
    assert.NotNil(t, evtBusTest.GetStoreManager())
    store := evtBusTest.GetStoreManager().CreateStore("test")
    assert.NotNil(t, store)
    assert.True(t, evtBusTest.GetStoreManager().DestroyStore("test"))
}

func TestChannelManager_TestConnectBroker(t *testing.T) {
    u := runWebSocketEndPoint()
    url, _ := url.Parse(u)
    host, port, _ := net.SplitHostPort(url.Host)
    testHost := host + ":" + port
    createTestChannel()

    // connect to broker
    cf := &bridge.BrokerConnectorConfig{
        Username:   "test",
        Password:   "test",
        UseWS: true,
        WSPath: "/",
        ServerAddr: testHost }
    c, _:= evtBusTest.ConnectBroker(cf)
    assert.NotNil(t, c)
    destroyTestChannel()
    killWebSocketEndpoint()
    util.ResetMonitor()
}

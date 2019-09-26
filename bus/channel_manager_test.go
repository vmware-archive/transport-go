// Copyright 2019 VMware Inc.

package bus

import (
    "go-bifrost/bridge"
    "go-bifrost/model"
    "go-bifrost/util"
    "net"
    "net/url"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
    "time"
)

var testChannelManager ChannelManager
var testChannelManagerChannelName string = "melody"

func createManager() ChannelManager {
    b := GetBus()
    manager := NewBusChannelManager(b)
    return manager
}

func TestChannelManager_Boot(t *testing.T) {
    testChannelManager = createManager()
    assert.Len(t, testChannelManager.GetAllChannels(), 0)
}

func TestChannelManager_CreateChannel(t *testing.T) {
    testChannelManager = createManager()

    testChannelManager.CreateChannel(testChannelManagerChannelName)
    assert.Len(t, testChannelManager.GetAllChannels(), 1)

    fetchedChannel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.NotNil(t, fetchedChannel)
    assert.True(t, testChannelManager.CheckChannelExists(testChannelManagerChannelName))
}

func TestChannelManager_GetNotExistentChannel(t *testing.T) {
    testChannelManager = createManager()

    fetchedChannel, err := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.NotNil(t, err)
    assert.Nil(t, fetchedChannel)
}

func TestChannelManager_DestroyChannel(t *testing.T) {
    testChannelManager = createManager()

    testChannelManager.CreateChannel(testChannelManagerChannelName)
    testChannelManager.DestroyChannel(testChannelManagerChannelName)
    fetchedChannel, err := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, testChannelManager.GetAllChannels(), 0)
    assert.NotNil(t, err)
    assert.Nil(t, fetchedChannel)
}

func TestChannelManager_SubscribeChannelHandler(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    uuid, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.Nil(t, err)
    assert.NotNil(t, uuid)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
}

func TestChannelManager_SubscribeChannelHandlerMissingChannel(t *testing.T) {
    testChannelManager = createManager()
    handler := func(*model.Message) {}
    _, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandler(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    uuid, _ := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)

    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, uuid)
    assert.Nil(t, err)
    assert.Len(t, channel.eventHandlers, 0)
}

func TestChannelManager_UnsubscribeChannelHandlerMissingChannel(t *testing.T) {
    testChannelManager = createManager()
    uuid := uuid.New()
    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, &uuid)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandlerNoId(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
    id := uuid.New()
    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, &id)
    assert.NotNil(t, err)
    assert.Len(t, channel.eventHandlers, 1)
}

func TestChannelManager_TestWaitForGroupOnBadChannel(t *testing.T) {
    testChannelManager = createManager()
    err := testChannelManager.WaitForChannel("unknown")
    assert.Error(t, err, "no such Channel as 'unknown'")
}

func TestChannelManager_TestGalacticChannelOpen(t *testing.T) {

    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)
    d := make(chan bool)
    s := util.GetMonitor()
    id := uuid.New()
    var listenMonitor = func() {
        // mark channel as galactic.

        c := &bridge.Connection{Id: &id}
        e := testChannelManager.MarkChannelAsGalactic(testChannelManagerChannelName, "/topic/testy-test", c)
        assert.Nil(t, e)

        for {
            e := <-s.Stream
            if e.EventType == util.ChannelIsGalacticEvt {
                evt := e.Message.Payload.(*galacticEvent)
                assert.Equal(t, "/topic/testy-test", evt.dest)
                d <- true
                break
            }
        }
    }
    go listenMonitor()
    // wait until we get what we need.
    <-d
    util.ResetMonitor()
}

func TestChannelManager_TestGalacticChannelOpenError(t *testing.T) {
    // channel is not open / does not exist, so this should fail.
    e := testChannelManager.MarkChannelAsGalactic(evtbusTestChannelName, "/topic/testy-test", nil)
    assert.Error(t, e)
    util.ResetMonitor()
}

func TestChannelManager_TestGalacticChannelCloseError(t *testing.T) {
    // channel is not open / does not exist, so this should fail.
    e := testChannelManager.MarkChannelAsLocal(evtbusTestChannelName)
    assert.Error(t, e)
    util.ResetMonitor()
}

func TestChannelManager_TestLocalChannel(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)
    d := make(chan bool)
    s := util.GetMonitor()
    var listenMonitor = func() {

        // map channel to galactic dest
        e := testChannelManager.MarkChannelAsLocal(testChannelManagerChannelName)
        assert.Nil(t, e)

        for {
            e := <-s.Stream
            if e.EventType == util.ChannelIsLocalEvt {
                d <- true
                break
            }
        }
    }

    go listenMonitor()
    // wait for us to get what we need.
    <-d
    util.ResetMonitor()
}

func TestChannelManager_TestListenToMonitorGalactic(t *testing.T) {
    util.ResetMonitor()
    myChan := "mychan"

    b := GetBus()

    // run ws and tcp brokers.
    u := runWebSocketEndPoint()

    url, _ := url.Parse(u)
    host, port, _ := net.SplitHostPort(url.Host)
    testHost := host + ":" + port

    testChannelManager = b.GetChannelManager()
    testChannelManager.ListenToMonitor()

    c := testChannelManager.CreateChannel(myChan)
    testChannelManager.ListenToMonitor()

    bc := bridge.NewBrokerConnector()
    cf := &bridge.BrokerConnectorConfig{Username: "guest", Password: "guest", UseWS: true, WSPath: "/", ServerAddr: testHost}
    conn, e := bc.Connect(cf)

    assert.Nil(t, e)
    assert.NotNil(t, conn)

    x := 0

    h, e := b.ListenOnce(myChan)
    assert.Nil(t, e)

    var m1 = make(chan bool)
    var m2 = make(chan bool)


    h.Handle(
        func(msg *model.Message) {
            x++
            m1 <- true
        },
        func(err error) {

        })

    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", conn)
    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", conn) // double up for fun
    <-c.brokerMappedEvent
    assert.Len(t, c.brokerConns, 1)
    <-m1
    go runStompBroker()

    // lets add another connection to the same channel.
    cf = &bridge.BrokerConnectorConfig{Username: "guest", Password: "guest", ServerAddr: testBrokerAddress}
    conn, e = bc.Connect(cf)

    assert.Nil(t, e)
    assert.NotNil(t, conn)

    h, e = b.ListenOnce(myChan)

    h.Handle(
        func(msg *model.Message) {
            x++
            m2 <- true
        },
        func(err error) {})

    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", conn)
    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", conn) // trigger double (should ignore)

    select {
    case <-c.brokerMappedEvent:
    case <-time.After(5 * time.Second):
        assert.FailNow(t, "TestChannelManager_TestListenToMonitorGalactic timeout on brokerMappedEvent")
    }

    err := conn.SendMessage("/queue/hiya",[]byte("Hi baby melody!"))
    assert.Nil(t, err)
    <-m2
    assert.Equal(t, 2, x)
}


// This test performs a end to end run of the monitor.
// it will create a ws broker subscription, map it to a single channel
// then it will unsubscribe and check that the unsubscription went through ok.
func TestChannelManager_TestListenToMonitorLocal(t *testing.T) {

    util.ResetMonitor()
    myChan := "mychan-local"

    b := GetBus()

    // run ws broker
    u := runWebSocketEndPoint()
    url, _ := url.Parse(u)
    host, port, _ := net.SplitHostPort(url.Host)
    testHost := host + ":" + port

    testChannelManager = b.GetChannelManager()
    testChannelManager.ListenToMonitor()

    c := testChannelManager.CreateChannel(myChan)

    bc := bridge.NewBrokerConnector()
    cf := &bridge.BrokerConnectorConfig{Username: "guest", Password: "guest", UseWS: true, WSPath: "/", ServerAddr: testHost}
    conn, e := bc.Connect(cf)

    assert.Nil(t, e)
    assert.NotNil(t, conn)

    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/seeya", conn)
    <-c.brokerMappedEvent
    assert.Len(t, c.brokerConns, 1)

    testChannelManager.MarkChannelAsLocal(myChan)
    <-c.brokerMappedEvent
    assert.Len(t, c.brokerConns, 0)
    assert.Len(t, c.brokerSubs, 0)
}

func TestChannelManager_TestGalacticMonitorInvalidChannel(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel("fun-chan")

    err := testChannelManager.MarkChannelAsGalactic("fun-chan", "/queue/woo", nil)
    assert.Nil(t, err)

}

func TestChannelManager_TestLocalMonitorInvalidChannel(t *testing.T) {
    testChannelManager = createManager()
    testChannelManager.CreateChannel("fun-chan")

    err := testChannelManager.MarkChannelAsLocal("fun-chan")
    assert.Nil(t, err)
}

func TestChannelManager_StopListeningMonitor(t *testing.T) {
    m := createManager().(*busChannelManager)
    m.CreateChannel("fun-chan")
    m.ListenToMonitor()
    assert.True(t, m.monitorActive)
    m.StopListeningMonitor()
    assert.False(t, m.monitorActive)
}

// Copyright 2019 VMware Inc.

package bus

import (
    "go-bifrost/model"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "testing"
    "time"
    "sync"
)

var testChannelManager ChannelManager
var testChannelManagerChannelName = "melody"

func createManager() (ChannelManager, EventBus) {
    b := newTestEventBus()
    manager := NewBusChannelManager(b)
    return manager, b
}

func TestChannelManager_Boot(t *testing.T) {
    testChannelManager, _ = createManager()
    assert.Len(t, testChannelManager.GetAllChannels(), 0)
}

func TestChannelManager_CreateChannel(t *testing.T) {
    var bus EventBus
    testChannelManager, bus = createManager()

    wg := sync.WaitGroup{}
    wg.Add(1)
    bus.AddMonitorEventListener(
        func(monitorEvt *MonitorEvent) {
            if monitorEvt.EntityName == testChannelManagerChannelName {
                assert.Equal(t, monitorEvt.EventType, ChannelCreatedEvt)
                wg.Done()
            }
        })

    testChannelManager.CreateChannel(testChannelManagerChannelName)

    wg.Wait()

    assert.Len(t, testChannelManager.GetAllChannels(), 1)

    fetchedChannel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.NotNil(t, fetchedChannel)
    assert.True(t, testChannelManager.CheckChannelExists(testChannelManagerChannelName))
}

func TestChannelManager_GetNotExistentChannel(t *testing.T) {
    testChannelManager, _ = createManager()

    fetchedChannel, err := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.NotNil(t, err)
    assert.Nil(t, fetchedChannel)
}

func TestChannelManager_DestroyChannel(t *testing.T) {
    testChannelManager, _ = createManager()

    testChannelManager.CreateChannel(testChannelManagerChannelName)
    testChannelManager.DestroyChannel(testChannelManagerChannelName)
    fetchedChannel, err := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, testChannelManager.GetAllChannels(), 0)
    assert.NotNil(t, err)
    assert.Nil(t, fetchedChannel)
}

func TestChannelManager_SubscribeChannelHandler(t *testing.T) {
    testChannelManager, _ = createManager()
    testChannelManager.CreateChannel(testChannelManagerChannelName)

    handler := func(*model.Message) {}
    uuid, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.Nil(t, err)
    assert.NotNil(t, uuid)
    channel, _ := testChannelManager.GetChannel(testChannelManagerChannelName)
    assert.Len(t, channel.eventHandlers, 1)
}

func TestChannelManager_SubscribeChannelHandlerMissingChannel(t *testing.T) {
    testChannelManager, _ = createManager()
    handler := func(*model.Message) {}
    _, err := testChannelManager.SubscribeChannelHandler(testChannelManagerChannelName, handler, false)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandler(t *testing.T) {
    testChannelManager, _ = createManager()
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
    testChannelManager, _ = createManager()
    uuid := uuid.New()
    err := testChannelManager.UnsubscribeChannelHandler(testChannelManagerChannelName, &uuid)
    assert.NotNil(t, err)
}

func TestChannelManager_UnsubscribeChannelHandlerNoId(t *testing.T) {
    testChannelManager, _ = createManager()
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
    testChannelManager, _ = createManager()
    err := testChannelManager.WaitForChannel("unknown")
    assert.Error(t, err, "no such Channel as 'unknown'")
}

func TestChannelManager_TestGalacticChannelOpen(t *testing.T) {

    testChannelManager, _ = createManager()
    galacticChannel := testChannelManager.CreateChannel(testChannelManagerChannelName)
    id := uuid.New()

    // mark channel as galactic.

    subId := uuid.New()
    sub := &MockBridgeSubscription{
        Id: &subId,
    }

    c := &MockBridgeConnection{Id: &id}
    c.On("Subscribe", "/topic/testy-test").Return(sub, nil).Once()
    e := testChannelManager.MarkChannelAsGalactic(testChannelManagerChannelName, "/topic/testy-test", c)

    assert.Nil(t, e)
    c.AssertExpectations(t)

    assert.True(t, galacticChannel.galactic)

    assert.Equal(t, len(galacticChannel.brokerConns), 1)
    assert.Equal(t, galacticChannel.brokerConns[0], c)

    assert.Equal(t, len(galacticChannel.brokerSubs), 1)
    assert.Equal(t, galacticChannel.brokerSubs[0].s, sub)
    assert.Equal(t, galacticChannel.brokerSubs[0].c, c)

    testChannelManager.MarkChannelAsLocal(testChannelManagerChannelName)
    assert.False(t, galacticChannel.galactic)

    assert.Equal(t, len(galacticChannel.brokerConns), 0)
    assert.Equal(t, len(galacticChannel.brokerSubs), 0)
}

func TestChannelManager_TestGalacticChannelOpenError(t *testing.T) {
    // channel is not open / does not exist, so this should fail.
    e := testChannelManager.MarkChannelAsGalactic(evtbusTestChannelName, "/topic/testy-test", nil)
    assert.Error(t, e)
}

func TestChannelManager_TestGalacticChannelCloseError(t *testing.T) {
    // channel is not open / does not exist, so this should fail.
    e := testChannelManager.MarkChannelAsLocal(evtbusTestChannelName)
    assert.Error(t, e)
}

func TestChannelManager_TestListenToMonitorGalactic(t *testing.T) {
    myChan := "mychan"

    b := newTestEventBus()

    testChannelManager = b.GetChannelManager()
    c := testChannelManager.CreateChannel(myChan)



    // mark channel as galactic.
    id := uuid.New()
    subId := uuid.New()
    mockSub := &MockBridgeSubscription{
        Id: &subId,
        Channel: make(chan *model.Message, 10),
        Destination: "/queue/hiya",
    }

    mockCon := &MockBridgeConnection{Id: &id}
    mockCon.On("Subscribe", "/queue/hiya").Return(mockSub, nil).Once()

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

    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", mockCon)
    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", mockCon) // double up for fun
    <-c.brokerMappedEvent
    assert.Len(t, c.brokerConns, 1)
    mockSub.GetMsgChannel() <- &model.Message{Payload: "test-message", Direction: model.ResponseDir}
    <-m1

    // lets add another connection to the same channel.

    id2 := uuid.New()
    subId2 := uuid.New()
    mockSub2 := &MockBridgeSubscription{
        Id: &subId2,
        Channel: make(chan *model.Message, 10),
        Destination: "/queue/hiya",
    }

    mockCon2 := &MockBridgeConnection{Id: &id2}
    mockCon2.On("Subscribe", "/queue/hiya").Return(mockSub2, nil).Once()

    h, e = b.ListenOnce(myChan)

    h.Handle(
        func(msg *model.Message) {
            x++
            m2 <- true
        },
        func(err error) {})

    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", mockCon2)
    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/hiya", mockCon2) // trigger double (should ignore)

    select {
    case <-c.brokerMappedEvent:
    case <-time.After(5 * time.Second):
        assert.FailNow(t, "TestChannelManager_TestListenToMonitorGalactic timeout on brokerMappedEvent")
    }

    mockSub.GetMsgChannel() <- &model.Message{Payload: "Hi baby melody!", Direction: model.ResponseDir}

    <-m2
    assert.Equal(t, 2, x)
}


// This test performs a end to end run of the monitor.
// it will create a ws broker subscription, map it to a single channel
// then it will unsubscribe and check that the unsubscription went through ok.
func TestChannelManager_TestListenToMonitorLocal(t *testing.T) {

    myChan := "mychan-local"

    b := newTestEventBus()

    // run ws broker
    testChannelManager = b.GetChannelManager()

    c := testChannelManager.CreateChannel(myChan)


    subId := uuid.New()
    sub := &MockBridgeSubscription{
        Id: &subId,
    }

    id := uuid.New()
    mockCon := &MockBridgeConnection{Id: &id}
    mockCon.On("Subscribe", "/queue/seeya").Return(sub, nil).Once()

    testChannelManager.MarkChannelAsGalactic(myChan, "/queue/seeya", mockCon)
    <-c.brokerMappedEvent
    assert.Len(t, c.brokerConns, 1)

    testChannelManager.MarkChannelAsLocal(myChan)
    <-c.brokerMappedEvent
    assert.Len(t, c.brokerConns, 0)
    assert.Len(t, c.brokerSubs, 0)
}

func TestChannelManager_TestGalacticMonitorInvalidChannel(t *testing.T) {
    testChannelManager, _ = createManager()
    testChannelManager.CreateChannel("fun-chan")

    err := testChannelManager.MarkChannelAsGalactic("fun-chan", "/queue/woo", nil)
    assert.Nil(t, err)

}

func TestChannelManager_TestLocalMonitorInvalidChannel(t *testing.T) {
    testChannelManager, _ = createManager()
    testChannelManager.CreateChannel("fun-chan")

    err := testChannelManager.MarkChannelAsLocal("fun-chan")
    assert.Nil(t, err)
}

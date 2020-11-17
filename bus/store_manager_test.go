// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "reflect"
    "sync"
    "sync/atomic"
    "testing"
)

func createTestStoreManager() StoreManager {
    return newStoreManager(GetBus())
}

func TestStoreManager_CreateStore(t *testing.T) {
    storeManager := createTestStoreManager()
    assert.NotNil(t, storeManager)

    wg := sync.WaitGroup{}

    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            store := storeManager.CreateStore("testStore")
            assert.NotNil(t, store)
            assert.Equal(t, store.GetName(), "testStore")
            assert.Equal(t, store, storeManager.GetStore("testStore"))
            wg.Done()
        }()
    }

    wg.Wait()

    store2 := storeManager.CreateStore("testStore2")
    assert.NotEqual(t, store2, storeManager.GetStore("testStore"))
}

func TestStoreManager_GetStore(t *testing.T) {
    storeManager := createTestStoreManager()
    storeManager.CreateStore("testStore")
    store := storeManager.GetStore("testStore")
    assert.Equal(t, store.GetName(), "testStore")

    assert.Nil(t, storeManager.GetStore("invalid-store"))
}

func TestStoreManager_DestroyStore(t *testing.T) {
    storeManager := createTestStoreManager()
    storeManager.CreateStore("testStore")

    var counter int32 = 0

    wg := sync.WaitGroup{}

    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            if storeManager.DestroyStore("testStore") {
                atomic.AddInt32(&counter, 1)
            }
            assert.Nil(t, storeManager.GetStore("testStore"))
            wg.Done()
        }()
    }

    wg.Wait()

    // Verify that only one of the DestroyStore calls was successful (has returned true)
    assert.Equal(t, counter, int32(1))
}

func TestStoreManager_ConfigureStoreSyncChannel(t *testing.T) {
    m := createTestStoreManager()
    id := uuid.New()
    con := &MockBridgeConnection{Id: &id}

    subId := uuid.New()
    s := &MockBridgeSubscription{
        Id: &subId,
    }
    syncChannelDst := "/topic-prefix/fabric-store-sync." + id.String()
    con.On("Subscribe", syncChannelDst).Return(s, nil)
    con.On("SendMessage", syncChannelDst, mock.Anything).Return(nil)
    m.ConfigureStoreSyncChannel(con, "/topic-prefix", "/pub-prefix")

    storeManagerImpl := m.(*storeManager)

    conf, ok := storeManagerImpl.syncChannels[id]
    assert.True(t, ok)
    assert.NotNil(t, conf.syncChannelName)
    assert.Equal(t, conf.pubPrefix, "/pub-prefix/")
    assert.Equal(t, conf.topicPrefix, "/topic-prefix/")
    assert.Equal(t, conf.conn, con)

    syncCh, _ := storeManagerImpl.eventBus.GetChannelManager().GetChannel(conf.syncChannelName)
    assert.True(t, syncCh.galactic)

    err := m.ConfigureStoreSyncChannel(con, "/topic-prefix", "/pub-prefix")
    assert.EqualError(t, err, "store sync channel already configured for this connection")
}

func TestStoreManager_OpenGalacticStore(t *testing.T) {
    m := createTestStoreManager()
    id := uuid.New()
    con := &MockBridgeConnection{Id: &id}

    var s BusStore
    var err error

    s, err = m.OpenGalacticStore("galacticStore", con)

    assert.Nil(t, s)
    assert.EqualError(t, err, "sync channel is not configured for this connection")

    subId := uuid.New()
    sub := &MockBridgeSubscription{
        Id: &subId,
    }
    con.On("Subscribe", mock.Anything).Return(sub, nil)
    con.On("SendMessage", mock.Anything, mock.Anything).Return(nil)
    m.ConfigureStoreSyncChannel(con, "/topic-prefix", "/pub-prefix")

    storeManagerImpl := m.(*storeManager)

    conf, _ := storeManagerImpl.syncChannels[id]
    assert.NotNil(t, conf)

    s, err = m.OpenGalacticStore("galacticStore", con)

    assert.Nil(t, err)
    assert.True(t, s.IsGalactic())

    wg := sync.WaitGroup{}
    wg.Add(1)

    s.WhenReady(func() {
        assert.Equal(t, len(s.AllValues()), 2)
        assert.Equal(t, s.GetValue("id1"), "value1")
        assert.Equal(t, s.GetValue("id2"), "value2")

        wg.Done()
    })

    var jsonBlob = []byte(`{
        "storeId": "galacticStore",
        "responseType": "storeContentResponse",
        "items": {
            "id1": "value1",
            "id2": "value2"
        } 
    }`)

    storeManagerImpl.eventBus.SendResponseMessage(conf.syncChannelName, jsonBlob, nil)
    wg.Wait()
}

type MockStoreItem struct {
    From string `json:"from"`
    Message string `json:"message"`
}

func TestStoreManager_OpenGalacticStoreWithType(t *testing.T) {
    m := createTestStoreManager()
    id := uuid.New()
    con := &MockBridgeConnection{Id: &id}

    subId := uuid.New()
    sub := &MockBridgeSubscription{
        Id: &subId,
    }
    con.On("Subscribe", mock.Anything).Return(sub, nil)
    con.On("SendMessage", mock.Anything, mock.Anything).Return(nil)
    m.ConfigureStoreSyncChannel(con, "/topic-prefix", "/pub-prefix")

    storeManagerImpl := m.(*storeManager)

    conf, _ := storeManagerImpl.syncChannels[id]
    assert.NotNil(t, conf)

    store,_ := m.OpenGalacticStoreWithItemType("galacticStore", con, reflect.TypeOf(MockStoreItem{}))

    store2 ,err := m.OpenGalacticStoreWithItemType("galacticStore", con, reflect.TypeOf(MockStoreItem{}))

    assert.Equal(t, store, store2)
    assert.Nil(t, err)

    assert.True(t, store.IsGalactic())

    wg := sync.WaitGroup{}
    wg.Add(1)

    store.WhenReady(func() {

        assert.Equal(t, len(store.AllValues()), 1)
        assert.Equal(t, store.GetValue("id1"), MockStoreItem{
            From: "test-user",
            Message: "test-message",
        })

        wg.Done()
    })

    var jsonBlob = []byte(`{
        "storeId": "galacticStore",
        "responseType": "storeContentResponse",
        "items": {
            "id1": {
                "from": "test-user",
                "message": "test-message"
            }
        } 
    }`)

    storeManagerImpl.eventBus.SendResponseMessage(conf.syncChannelName, jsonBlob, nil)
    wg.Wait()
}

func TestStoreManager_OpenGalacticStoreWithLocalStoreId(t *testing.T) {

    m := createTestStoreManager()
    id := uuid.New()
    con := &MockBridgeConnection{Id: &id}
    subId := uuid.New()
    sub := &MockBridgeSubscription{
        Id: &subId,
    }
    con.On("Subscribe", mock.Anything).Return(sub, nil)
    con.On("SendMessage", mock.Anything, mock.Anything).Return(nil)
    m.ConfigureStoreSyncChannel(con, "/topic-prefix", "/pub-prefix")


    localStore := m.CreateStore("localStore")

    store, err := m.OpenGalacticStoreWithItemType("localStore", con, reflect.TypeOf(MockStoreItem{}))

    assert.EqualError(t, err, "cannot open galactic store: there is a local store with the same name")
    assert.Equal(t, store, localStore)
}

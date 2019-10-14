// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package bus

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "sync"
    "sync/atomic"
    "go-bifrost/bridge"
    "github.com/google/uuid"
    "reflect"
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
    con := &bridge.Connection{Id: &id}
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
    con := &bridge.Connection{Id: &id}

    var s BusStore
    var err error

    s, err = m.OpenGalacticStore("galacticStore", con)

    assert.Nil(t, s)
    assert.EqualError(t, err, "sync channel is not configured for this connection")


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
    con := &bridge.Connection{Id: &id}

    m.ConfigureStoreSyncChannel(con, "/topic-prefix", "/pub-prefix")

    storeManagerImpl := m.(*storeManager)

    conf, _ := storeManagerImpl.syncChannels[id]
    assert.NotNil(t, conf)

    store,_ := m.OpenGalacticStoreWithItemType("galacticStore", con, reflect.TypeOf(MockStoreItem{}))

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

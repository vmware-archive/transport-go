package store

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "sync"
    "sync/atomic"
    "fmt"
)

type testItem struct {
    name      string
    nameIndex int32
}

func testStore() BusStore {
    store := newBusStore("testStore")
    store.Initialize()
    return store
}

func TestBusStore_CreateStore(t *testing.T) {
    store := testStore()
    assert.Equal(t, store.GetName(), "testStore")
}

func TestBusStore_PutAndGet(t *testing.T) {
    store := testStore()
    store.Put("id1", 1, "ITEM_ADDED")
    store.Put("id2", "value2", "ITEM_ADDED")
    store.Put("id3", nil, "ITEM_ADDED")

    v, ok := store.Get("id1")
    assert.Equal(t, v, 1)
    assert.True(t, ok)

    v, _ = store.Get("id2")
    assert.Equal(t, v, "value2")

    v, ok = store.Get("id3")
    assert.Equal(t, v, nil)
    assert.True(t, ok)

    v, ok = store.Get("invalid-id")
    assert.False(t, ok)
    assert.Nil(t, v)
}

func TestBusStore_Remove(t *testing.T) {
    store := testStore()
    store.Put("id1", "item1", "ITEM_ADDED")

    var mutationEventsCounter int32 = 0
    var successfulRemovesCounter int32 = 0

    wg := sync.WaitGroup{}
    wg.Add(1)

    stream := store.OnAllChanges("ITEM_REMOVED")
    stream.Subscribe(func(change *StoreChange) {
        atomic.AddInt32(&mutationEventsCounter, 1)
        wg.Done()
    })

    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            if store.Remove("id1", "ITEM_REMOVED") {
                atomic.AddInt32(&successfulRemovesCounter, 1)
            }
            wg.Done()
        }()
    }

    wg.Wait()

    assert.False(t, store.Remove("invalid-id", "ITEM_REMOVED"))

    // Verify that only one of the Remove calls was successful (has returned true)
    assert.Equal(t, successfulRemovesCounter, int32(1))
    assert.Equal(t, mutationEventsCounter, int32(1))
}

func TestBusStore_AllValuesAndAllValuesAsMap(t *testing.T) {
    store := testStore()

    items := store.AllValues()
    allItemsAsMap := store.AllValuesAsMap()
    assert.Equal(t, len(items), 0)
    assert.Equal(t, len(allItemsAsMap), 0)


    store.Put("id1", testItem { name: "item1", nameIndex: 1}, "ITEM_ADDED")
    store.Put("id2", testItem { name: "item2", nameIndex: 2}, "ITEM_ADDED")
    store.Put("id3", testItem { name: "item3", nameIndex: 3}, "ITEM_ADDED")

    items = store.AllValues()
    allItemsAsMap = store.AllValuesAsMap()

    assert.Equal(t, len(items), 3)
    for _, item := range items {
        assert.Equal(t, fmt.Sprintf("item%d", item.(testItem).nameIndex), item.(testItem).name)
    }

    assert.Equal(t, len(allItemsAsMap), 3)
    assert.Equal(t, allItemsAsMap["id1"], testItem { name: "item1", nameIndex: 1})
    assert.Equal(t, allItemsAsMap["id2"], testItem { name: "item2", nameIndex: 2})
    assert.Equal(t, allItemsAsMap["id3"], testItem { name: "item3", nameIndex: 3})
}

func TestBusStore_OnChange(t *testing.T) {
    store := testStore()

    wg := sync.WaitGroup{}

    allChangesStreams := make([]StoreStream, 0)

    var allChangesCounter int32 = 0
    for i := 0; i < 5; i++ {
        stream := store.OnChange("id1")
        allChangesStreams = append(allChangesStreams, stream)
        stream.Subscribe(func(change *StoreChange) {
            atomic.AddInt32(&allChangesCounter, 1)
            wg.Done()
        })
    }

    var itemUpdateCounter int32 = 0
    for i := 0; i < 5; i++ {
        store.OnChange("id1", "ITEM_REMOVE", "ITEM_UPDATE").Subscribe(
            func(change *StoreChange) {
                atomic.AddInt32(&itemUpdateCounter, 1)
                wg.Done()
            })
    }

    for i := 0; i < 200; i++ {
        if i % 2 == 0 {
            wg.Add(10)
            go func() {
                store.Put("id1", "newValue", "ITEM_UPDATE")
            }()
        } else {
            wg.Add(5)
            go func() {
                store.Put("id1", "newValue", "ITEM_ADD")
            }()
        }
    }

    wg.Wait()

    assert.Equal(t, allChangesCounter, int32(1000))
    assert.Equal(t, itemUpdateCounter, int32(500))

    // Unsubscribe all changes listeners
    for _, stream := range allChangesStreams {
        stream.Unsubscribe()
    }

    wg.Add(5)
    store.Put("id1", "newValue", "ITEM_REMOVE")
    wg.Wait()

    assert.Equal(t, allChangesCounter, int32(1000))
    assert.Equal(t, itemUpdateCounter, int32(505))
}

func TestBusStore_OnChangeErrorHandling(t *testing.T) {
    store := testStore()
    stream := store.OnChange("id1")
    e := stream.Unsubscribe()
    assert.EqualError(t, e, "stream not subscribed")

    subscribeErr := stream.Subscribe(func(change *StoreChange) {
    })

    assert.Nil(t, subscribeErr)
    subscribeErr = stream.Subscribe(func(change *StoreChange) {
    })
    assert.EqualError(t, subscribeErr, "stream already subscribed")

    e = stream.Subscribe(nil)
    assert.EqualError(t, e, "invalid StoreChangeHandlerFunction")
}

func TestBusStore_OnAllChanges(t *testing.T) {
    store := testStore()

    wg := sync.WaitGroup{}

    var allChangesCounter int32 = 0
    allChangesStream := store.OnAllChanges()
    allChangesStream.Subscribe(func(change *StoreChange) {
        atomic.AddInt32(&allChangesCounter, 1)
        wg.Done()
    })

    itemUpdatedStream := store.OnAllChanges("ITEM_UPDATED", "ITEM_REMOVED")
    var itemUpdateCounter int32 = 0
    itemUpdatedStream.Subscribe(
        func(change *StoreChange) {
            atomic.AddInt32(&itemUpdateCounter, 1)
            wg.Done()
        })


    for i := 0; i < 200; i++ {
        if i % 2 == 0 {
            wg.Add(2)
            go func() {
                store.Put("id1", "newValue", "ITEM_UPDATED")
            }()
        } else {
            wg.Add(1)
            go func() {
                store.Put("id1", "newValue", "ITEM_ADD")
            }()
        }
    }

    wg.Wait()

    assert.Equal(t, allChangesCounter, int32(200))
    assert.Equal(t, itemUpdateCounter, int32(100))

    allChangesStream.Unsubscribe()

    wg.Add(1)
    store.Put("id1", "newValue", "ITEM_REMOVED")
    wg.Wait()

    assert.Equal(t, allChangesCounter, int32(200))
    assert.Equal(t, itemUpdateCounter, int32(101))
}

func TestBusStore_WhenReady(t *testing.T) {
    store := newBusStore("testStore")

    wg := sync.WaitGroup{}
    var counter int32 = 0
    for i := 0; i < 100; i++ {
        wg.Add(1)
        store.WhenReady(func() {
            atomic.AddInt32(&counter, 1)
            wg.Done()
        })
    }

    store.Initialize()

    wg.Wait()
    assert.Equal(t, counter, int32(100))

    for i := 0; i < 100; i++ {
        wg.Add(1)
        store.WhenReady(func() {
            atomic.AddInt32(&counter, 1)
            wg.Done()
        })
    }

    wg.Wait()
    assert.Equal(t, counter, int32(200))
}

func TestBusStore_Populate(t *testing.T) {
    store := newBusStore("testStore")

    wg := sync.WaitGroup{}
    counter := 0

    wg.Add(1)
    store.WhenReady(func() {
        counter++
        wg.Done()
    })

    err := store.Populate(map[string]interface{} {
        "id1":  1,
        "id2":  2,
        "id3":  3,
        "id4":  4,
    })

    assert.Nil(t, err)

    wg.Wait()

    assert.Equal(t, counter, 1)

    allValues := store.AllValuesAsMap()
    assert.Equal(t, len(allValues), 4)
    assert.Equal(t, allValues["id1"], 1)
    assert.Equal(t, allValues["id2"], 2)
    assert.Equal(t, allValues["id3"], 3)
    assert.Equal(t, allValues["id4"], 4)

    err = store.Populate(map[string]interface{} {
        "id1":  1,
    })

    assert.EqualError(t, err, "store items already initialized")
    assert.Equal(t, len(store.AllValues()), 4)
}

func TestBusStore_Reset(t *testing.T) {
    store := newBusStore("testStore")
    wg := sync.WaitGroup{}
    counter := 0

    wg.Add(1)
    store.WhenReady(func() {
        counter++
        wg.Done()
    })

    store.Populate(map[string]interface{} {
        "id1":  1,
        "id2":  2,
        "id3":  3,
    })
    wg.Wait()

    store.Reset()

    assert.Equal(t, len(store.AllValues()), 0)

    wg.Add(1)
    store.WhenReady(func() {
        counter++
        wg.Done()
    })

    store.Initialize()
    wg.Wait()
    assert.Equal(t, counter, 2)
}

func TestBusStore_OnMutationRequest(t *testing.T) {
    store := testStore()

    var allMutationsCounter int32 = 0
    var responseCount int32 = 0
    var errorCount int32 = 0

    allMutationsStream := store.OnMutationRequest()
    allMutationsStream.Subscribe(func(mutationReq *MutationRequest) {
        atomic.AddInt32(&allMutationsCounter, 1)
        mutationReq.SuccessHandler( mutationReq.Request.(string) + "-response")
    })

    var updateMutationsCounter int32 = 0
    updateMutationStream := store.OnMutationRequest("UPDATE_ITEM", "REMOVE_ITEM")
    updateMutationStream.Subscribe(func(mutationReq *MutationRequest) {
        atomic.AddInt32(&updateMutationsCounter, 1)
        mutationReq.ErrorHandler(mutationReq.Request.(string) + "-error")
    })

    wg := sync.WaitGroup{}

    for i := 0; i < 100; i++ {
        req := fmt.Sprintf("req%d", i)
        wg.Add(2)
        go func() {
            store.Mutate(req, "UPDATE_ITEM",
                func(result interface{}) {
                    assert.Equal(t, result, req + "-response")
                    atomic.AddInt32(&responseCount, 1)
                    wg.Done()
                },
                func(err interface{}) {
                    assert.Equal(t, err, req + "-error")
                    atomic.AddInt32(&errorCount, 1)
                    wg.Done()
                })
        }()

        wg.Add(1)
        go func() {
            store.Mutate(req, "MODIFY_ITEM",
                func(result interface{}) {
                    assert.Equal(t, result, req + "-response")
                    atomic.AddInt32(&responseCount, 1)
                    wg.Done()
                },
                nil)
        }()
    }

    wg.Wait()
    assert.Equal(t, allMutationsCounter, int32(200))
    assert.Equal(t, responseCount, int32(200))
    assert.Equal(t, updateMutationsCounter, int32(100))
    assert.Equal(t, errorCount, int32(100))

    allMutationsStream.Unsubscribe()

    wg.Add(1)
    store.Mutate("req", "UPDATE_ITEM",
        nil,
        func(err interface{}) {
            assert.Equal(t, err, "req-error")
            atomic.AddInt32(&errorCount, 1)
            wg.Done()
        })
    wg.Wait()
    assert.Equal(t, allMutationsCounter, int32(200))
    assert.Equal(t, responseCount, int32(200))
    assert.Equal(t, updateMutationsCounter, int32(101))
    assert.Equal(t, errorCount, int32(101))
}

func TestBusStore_OnMutationRequest_ErrorHandling(t *testing.T) {
    store := testStore()

    ms := store.OnMutationRequest()
    err := ms.Unsubscribe()

    assert.EqualError(t, err, "stream not subscribed")

    err = ms.Subscribe(nil)
    assert.EqualError(t, err, "invalid MutationRequestHandlerFunction")

    ms.Subscribe(func(mutationReq *MutationRequest) {})

    err = ms.Subscribe(func(mutationReq *MutationRequest) {})
    assert.EqualError(t, err, "stream already subscribed")
}
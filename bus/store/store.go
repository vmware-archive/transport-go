// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package store

import (
    "sync"
    "fmt"
)

// Describes a single store item change
type StoreChange struct {
    Id              string      // the id of the updated item
    Value           interface{} // the updated value of the item
    State           interface{} // state associated with this change
    IsDeleteChange  bool        // true if the item was removed from the store
    StoreVersion    int64       // the store's version when this change was made
}

// BusStore is a stateful in memory cache for objects. All state changes (any time the cache is modified)
// will broadcast that updated object to any subscribers of the BusStore for those specific objects
// or all objects of a certain type and state changes.
type BusStore interface {
    // Get the name (the id) of the store.
    GetName() string
    // Add new or updates existing item in the store.
    Put(id string, value interface{}, state interface{})
    // Get an item from the store.
    Get(id string) (interface{}, bool)
    // Remove an item from the store. Returns true if the remove operation was successful.
    Remove(id string, state interface{}) bool
    // Return a slice containing all store items.
    AllValues() []interface{}
    // Return a map with all items from the store.
    AllValuesAsMap() map[string]interface{}
    // Subscribe to state changes for a specific object.
    OnChange(id string, state ...interface{}) StoreStream
    // Subscribe to state changes for all objects
    OnAllChanges(state ...interface{}) StoreStream
    // Notify when the store has been initialize (via populate() or initialize()
    WhenReady(readyFunction func())
    // Populate the store with a map of items and their ID's.
    Populate(items map[string]interface{}) error
    // Mark the store as initialized and notify all watchers.
    Initialize()
    // Subscribe to mutation requests made via mutate() method.
    OnMutationRequest(mutationType ...interface{}) MutationStoreStream
    // Send a mutation request to any subscribers handling mutations.
    Mutate(request interface{}, requestType interface{},
            successHandler func(interface{}), errorHandler func(interface{}))
    // Removes all items from the store and change its state to uninitialized".
    Reset()
}

// Internal BusStore implementation
type busStore struct {
    name                 string
    itemsLock            sync.RWMutex
    items                map[string]interface{}
    storeVersion         int64
    storeStreamsLock     sync.RWMutex
    storeStreams         []*storeStream
    mutationStreamsLock  sync.RWMutex
    mutationStreams      []*mutationStoreStream
    initializer          sync.Once
    readyC               chan struct{}
}

func newBusStore(name string) BusStore {
    store := new(busStore)
    store.name = name

    initStore(store)
    return store
}

func initStore(store *busStore) {
    store.readyC = make(chan struct{})
    store.storeStreams = []*storeStream {}
    store.items = make(map[string]interface{})
    store.storeVersion = 1
    store.initializer = sync.Once{}
}

func (store *busStore) GetName() string {
    return store.name
}

func (store *busStore) Populate(items map[string]interface{}) error {
    store.itemsLock.Lock()
    defer store.itemsLock.Unlock()

    if len(store.items) > 0 {
        return fmt.Errorf("store items already initialized")
    }

    for k,v := range items {
        store.items[k] = v
    }
    store.Initialize()

    return nil
}

func (store *busStore) Put(id string, value interface{}, state interface{}) {
    store.itemsLock.Lock()
    defer store.itemsLock.Unlock()

    store.storeVersion++
    store.items[id] = value

    change := &StoreChange{
        Id: id,
        State: state,
        Value: value,
        StoreVersion: store.storeVersion,
    }

    go store.onStoreChange(change)
}

func (store *busStore) Get(id string) (interface{}, bool) {
    store.itemsLock.RLock()
    defer store.itemsLock.RUnlock()

    val, ok := store.items[id]

    return val, ok
}

func (store *busStore) Remove(id string, state interface{}) bool {
    store.itemsLock.Lock()
    defer store.itemsLock.Unlock()

    value, ok := store.items[id]
    if !ok {
        return false
    }

    store.storeVersion++
    delete(store.items, id)

    change := &StoreChange{
        Id: id,
        State: state,
        Value: value,
        StoreVersion: store.storeVersion,
        IsDeleteChange: true,
    }

    go store.onStoreChange(change)
    return true
}

func (store *busStore) AllValues() []interface{} {

    store.itemsLock.RLock()
    defer store.itemsLock.RUnlock()

    values := make([] interface{}, 0, len(store.items))
    for _, value := range store.items {
        values = append(values, value)
    }

    return values
}

func (store *busStore) AllValuesAsMap() map[string]interface{} {
    store.itemsLock.RLock()
    defer store.itemsLock.RUnlock()

    values := make(map[string] interface{})

    for key, value := range store.items {
        values[key] = value
    }

    return values
}

func (store *busStore) OnMutationRequest(requestType ...interface{}) MutationStoreStream {
    return newMutationStoreStream(store, &mutationStreamFilter{
        requestTypes: requestType,
    })
}

func (store *busStore) Mutate(request interface{}, requestType interface{},
        successHandler func(interface{}), errorHandler func(interface{})) {

    store.mutationStreamsLock.RLock()
    defer store.mutationStreamsLock.RUnlock()

    for _, ms := range store.mutationStreams {
        ms.onMutationRequest(&MutationRequest{
            Request: request,
            RequestType: requestType,
            SuccessHandler: successHandler,
            ErrorHandler: errorHandler,
        })
    }
}

func(store *busStore) onStoreChange(change *StoreChange) {
    store.storeStreamsLock.RLock()
    defer store.storeStreamsLock.RUnlock()

    for _, storeStream := range store.storeStreams {
        storeStream.onStoreChange(change)
    }
}

func (store *busStore) Initialize() {
    store.initializer.Do(func() {
        close(store.readyC)
    })
}

func (store *busStore) Reset() {
    store.itemsLock.Lock()
    defer store.itemsLock.Unlock()

    store.mutationStreamsLock.Lock()
    defer store.mutationStreamsLock.Unlock()

    store.storeStreamsLock.Lock()
    defer store.storeStreamsLock.Unlock()

    initStore(store)
}

func (store *busStore) WhenReady(readyFunc func()) {
    go func() {
        <- store.readyC
        readyFunc()
    }()
}

func (store *busStore) OnChange(id string, state ...interface{}) StoreStream {
    return newStoreStream(store, &streamFilter{
        itemId: id,
        states: state,
    })
}

func (store *busStore) OnAllChanges(state ...interface{}) StoreStream {
    return newStoreStream(store, &streamFilter{
        states: state,
        matchAllItems: true,
    })
}

func (store *busStore) onStreamSubscribe(stream *storeStream) {
    store.storeStreamsLock.Lock()
    defer store.storeStreamsLock.Unlock()

    store.storeStreams = append(store.storeStreams, stream)
}

func (store *busStore) onMutationStreamSubscribe(stream *mutationStoreStream) {
    store.mutationStreamsLock.Lock()
    defer store.mutationStreamsLock.Unlock()

    store.mutationStreams = append(store.mutationStreams, stream)
}

func (store *busStore) onStreamUnsubscribe(stream *storeStream) {
    store.storeStreamsLock.Lock()
    defer store.storeStreamsLock.Unlock()

    var i int
    var s *storeStream
    for i, s = range store.storeStreams {
        if s == stream {
            break
        }
    }

    if s == stream {
        n := len(store.storeStreams)
        store.storeStreams[i] = store.storeStreams[n-1]
        store.storeStreams = store.storeStreams[:n-1]
    }
}

func (store *busStore) onMutationStreamUnsubscribe(stream *mutationStoreStream) {
    store.mutationStreamsLock.Lock()
    defer store.mutationStreamsLock.Unlock()

    var i int
    var s *mutationStoreStream
    for i, s = range store.mutationStreams {
        if s == stream {
            break
        }
    }

    if s == stream {
        n := len(store.mutationStreams)
        store.mutationStreams[i] = store.mutationStreams[n-1]
        store.mutationStreams = store.mutationStreams[:n-1]
    }
}

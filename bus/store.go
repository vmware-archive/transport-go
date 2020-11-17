// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
    "github.com/vmware/transport-go/log"
    "github.com/vmware/transport-go/model"
    "reflect"
    "sync"
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
    // Returns an item from the store and a boolean flag
    // indicating whether the item exists
    Get(id string) (interface{}, bool)
    // Shorten version of the Get() method, returns only the item value.
    GetValue(id string) interface{}
    // Remove an item from the store. Returns true if the remove operation was successful.
    Remove(id string, state interface{}) bool
    // Return a slice containing all store items.
    AllValues() []interface{}
    // Return a map with all items from the store.
    AllValuesAsMap() map[string]interface{}
    // Return a map with all items from the store with the current store version.
    AllValuesAndVersion() (map[string]interface{}, int64)
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
    // Returns true if this is galactic store.
    IsGalactic() bool
    // Get the item type if such is specified during the creation of the
    // store
    GetItemType() reflect.Type
}

// Internal BusStore implementation
type busStore struct {
    name                string
    itemsLock           sync.RWMutex
    items               map[string]interface{}
    storeVersion        int64
    storeStreamsLock    sync.RWMutex
    storeStreams        []*storeStream
    mutationStreamsLock sync.RWMutex
    mutationStreams     []*mutationStoreStream
    initializer         sync.Once
    readyC              chan struct{}
    isGalactic          bool
    galacticConf        *galacticStoreConfig
    bus                 EventBus
    itemType            reflect.Type
    storeSynHandler     MessageHandler
}

type galacticStoreConfig struct {
    syncChannelConfig   *storeSyncChannelConfig
}

func newBusStore(name string, bus EventBus, itemType reflect.Type, galacticConf *galacticStoreConfig) BusStore {

    store := new(busStore)
    store.name = name
    store.bus = bus
    store.itemType = itemType
    store.galacticConf = galacticConf

    initStore(store)

    store.isGalactic = galacticConf != nil

    if store.isGalactic {
        initGalacticStore(store)
    }

    return store
}

func initStore(store *busStore) {
    store.readyC = make(chan struct{})
    store.storeStreams = []*storeStream {}
    store.mutationStreams = []*mutationStoreStream {}
    store.items = make(map[string]interface{})
    store.storeVersion = 1
    store.initializer = sync.Once{}
}

func initGalacticStore(store *busStore) {

    syncChannelConf := store.galacticConf.syncChannelConfig

    var err error
    store.storeSynHandler, err = store.bus.ListenStream(syncChannelConf.syncChannelName)
    if err != nil {
        return
    }

    store.storeSynHandler.Handle(
        func(msg *model.Message) {
            d := msg.Payload.([]byte)
            var storeResponse map[string]interface{}

            err := json.Unmarshal(d, &storeResponse)
            if err != nil {
                log.Warn("failed to unmarshal storeResponse")
                return
            }

            if storeResponse["storeId"] != store.GetName() {
                // the response is for another store
                return
            }

            responseType := storeResponse["responseType"].(string)

            switch responseType {
            case "storeContentResponse":

                store.itemsLock.Lock()
                defer store.itemsLock.Unlock()

                store.updateVersionFromResponse(storeResponse)
                items := storeResponse["items"].(map[string]interface{})
                store.items = make(map[string]interface{})
                for key, val := range items {
                    deserializedValue, err :=  store.deserializeRawValue(val)
                    if err != nil {
                        log.Warn("failed to deserialize store item value %e", err)
                        continue
                    } else {
                        store.items[key] = deserializedValue
                    }
                }
                store.Initialize()
            case "updateStoreResponse":

                store.itemsLock.Lock()
                defer store.itemsLock.Unlock()

                store.updateVersionFromResponse(storeResponse)
                newItemRaw, ok := storeResponse["newItemValue"]
                itemId := storeResponse["itemId"].(string)
                if !ok || newItemRaw == nil {
                    store.removeInternal(itemId, "galacticSyncRemove")
                } else {
                    newItemValue, err := store.deserializeRawValue(newItemRaw)
                    if err != nil {
                        log.Warn("failed to deserialize store item value %e", err)
                        return
                    }
                    store.putInternal(itemId, newItemValue, "galacticSyncUpdate")
                }
            }
        },
        func(e error) {
        })

    store.sendOpenStoreRequest()
}

func (store *busStore) updateVersionFromResponse(storeResponse map[string]interface{}) {
    version := storeResponse["storeVersion"]
    switch version.(type) {
    case float64:
        store.storeVersion = int64(version.(float64))
    case int64:
        store.storeVersion = version.(int64)
    default:
        log.Warn("failed to deserialize store version")
        store.storeVersion = 1
    }
}

func (store *busStore) deserializeRawValue(rawValue interface{}) (interface{}, error) {
    return model.ConvertValueToType(rawValue, store.itemType)
}

func (store *busStore) sendOpenStoreRequest() {
    openStoreReq := map[string]string {
        "storeId": store.GetName(),
    }
    store.sendGalacticRequest("openStore", openStoreReq)
}

func (store *busStore) sendGalacticRequest(requestCmd string, requestPayload interface{}) {
    // create request
    id := uuid.New();
    r := &model.Request{}
    r.Request = requestCmd
    r.Payload = requestPayload
    r.Id = &id
    jsonReq, _ := json.Marshal(r)

    syncChannelConfig := store.galacticConf.syncChannelConfig

    // send request.
    syncChannelConfig.conn.SendMessage(
        syncChannelConfig.pubPrefix + syncChannelConfig.syncChannelName,
        jsonReq)
}

func (store *busStore) sendCloseStoreRequest() {
    closeStoreReq := map[string]string {
        "storeId": store.GetName(),
    }
    store.sendGalacticRequest("closeStore", closeStoreReq)
}

func (store *busStore) OnDestroy() {
    if store.IsGalactic() {
        store.sendCloseStoreRequest()
        if store.storeSynHandler != nil {
            store.storeSynHandler.Close()
        }
    }
}

func (store *busStore) IsGalactic() bool {
    return store.isGalactic
}

func (store *busStore) GetItemType() reflect.Type {
    return store.itemType
}

func (store *busStore) GetName() string {
    return store.name
}

func (store *busStore) Populate(items map[string]interface{}) error {
    if store.IsGalactic() {
            return fmt.Errorf("populate() API is not supported for galactic stores")
    }

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
    if store.IsGalactic() {
        store.putGalactic(id, value)
    } else {
        store.itemsLock.Lock()
        defer store.itemsLock.Unlock()

        store.putInternal(id, value, state)
    }
}

func (store *busStore) putGalactic(id string, value interface{}) {
    store.itemsLock.RLock()
    clientStoreVersion := store.storeVersion
    store.itemsLock.RUnlock()

    store.sendUpdateStoreRequest(id, value, clientStoreVersion)
}

func (store *busStore) sendUpdateStoreRequest(id string, value interface{}, storeVersion int64) {
    updateReq := map[string]interface{} {
        "storeId": store.GetName(),
        "clientStoreVersion": storeVersion,
        "itemId": id,
        "newItemValue": value,
    }

    store.sendGalacticRequest("updateStore", updateReq)
}

func (store *busStore) putInternal(id string, value interface{}, state interface{}) {
    if !store.IsGalactic() {
        store.storeVersion++
    }
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

func (store *busStore) GetValue(id string) interface{} {
    val, _ := store.Get(id)
    return val
}

func (store *busStore) Remove(id string, state interface{}) bool {
    if store.IsGalactic() {
        return store.removeGalactic(id)
    } else {
        store.itemsLock.Lock()
        defer store.itemsLock.Unlock()

        return store.removeInternal(id, state)
    }
}

func (store *busStore) removeGalactic(id string) bool {
    store.itemsLock.RLock()
    _, ok := store.items[id]
    storeVersion := store.storeVersion
    store.itemsLock.RUnlock()

    if ok {
        store.sendUpdateStoreRequest(id, nil, storeVersion)
        return true
    }
    return false
}

func (store *busStore) removeInternal(id string, state interface{}) bool {
    value, ok := store.items[id]
    if !ok {
        return false
    }

    if !store.IsGalactic() {
        store.storeVersion++
    }
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

func (store *busStore) AllValuesAndVersion() (map[string]interface{}, int64) {
    store.itemsLock.RLock()
    defer store.itemsLock.RUnlock()

    values := make(map[string] interface{})

    for key, value := range store.items {
        values[key] = value
    }

    return values, store.storeVersion
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
        store.bus.SendMonitorEvent(StoreInitializedEvt, store.name, nil)
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

    if (store.IsGalactic()) {
        store.sendOpenStoreRequest()
    }
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

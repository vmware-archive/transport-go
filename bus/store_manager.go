// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package bus

import (
    "sync"
    "go-bifrost/bridge"
    "github.com/google/uuid"
    "fmt"
    "strings"
    "reflect"
)

// StoreManager interface controls all access to BusStores
type StoreManager interface {
    // Create a new Store, if the store already exists, then it will be returned.
    CreateStore(name string) BusStore
    // Get a reference to the existing store. Returns nil if the store doesn't exist.
    GetStore(name string) BusStore
    // Deletes a store.
    DestroyStore(name string) bool
    // Configure galactic store sync channel for a given connection.
    // Should be called before OpenGalacticStore() and OpenGalacticStoreWithItemType() APIs.
    ConfigureStoreSyncChannel(conn *bridge.Connection, topicPrefix string, pubPrefix string) error
    // Open new galactic store
    OpenGalacticStore(name string, conn *bridge.Connection) (BusStore, error)
    // Open new galactic store and deserialize items from server to itemType
    OpenGalacticStoreWithItemType(name string, conn *bridge.Connection, itemType reflect.Type) (BusStore, error)
}

// Interface which is a subset of the bridge.Connection methods.
// Used to mock connection objects during unit testing.
type galacticStoreConnection interface {
    SendMessage(destination string, payload []byte) error
}

type storeSyncChannelConfig struct {
    topicPrefix     string
    pubPrefix       string
    syncChannelName string
    conn            galacticStoreConnection
}

type storeManager struct {
    stores           map[string]BusStore
    storesLock       sync.RWMutex
    eventBus         EventBus
    syncChannelsLock sync.RWMutex
    syncChannels     map[uuid.UUID]*storeSyncChannelConfig
}

func newStoreManager(eventBus EventBus) StoreManager {
    m := new(storeManager)
    m.stores = make(map[string] BusStore)
    m.syncChannels = make(map[uuid.UUID]*storeSyncChannelConfig)
    m.eventBus = eventBus

    return m
}

func (m *storeManager) CreateStore(name string) BusStore {
    m.storesLock.Lock()
    defer m.storesLock.Unlock()

    store, ok := m.stores[name]

    if ok {
        return store
    }

    m.stores[name] = newBusStore(name, m.eventBus, nil)
    return m.stores[name]
}

func (m *storeManager) GetStore(name string) BusStore {
    m.storesLock.RLock()
    defer m.storesLock.RUnlock()

    return m.stores[name]
}

func (m *storeManager) DestroyStore(name string) bool {
    m.storesLock.Lock()
    defer m.storesLock.Unlock()

    store, ok := m.stores[name]
    if ok {
        store.(*busStore).OnDestroy()

        delete(m.stores, name)
    }
    return ok
}

func (m *storeManager) ConfigureStoreSyncChannel(
        conn *bridge.Connection, topicPrefix string, pubPrefix string) error {

    m.syncChannelsLock.Lock()
    defer m.syncChannelsLock.Unlock()

    _, ok := m.syncChannels[*conn.Id]
    if ok {
        return fmt.Errorf("store sync channel already configured for this connection")
    }

    if !strings.HasSuffix(topicPrefix, "/") {
        topicPrefix += "/"
    }
    if !strings.HasSuffix(pubPrefix, "/") {
        pubPrefix += "/"
    }

    syncChannel := "fabric-store-sync." + conn.Id.String()

    storeSyncChannelConfig := &storeSyncChannelConfig{
        topicPrefix:     topicPrefix,
        pubPrefix:       pubPrefix,
        syncChannelName: syncChannel,
        conn:            conn,
    }
    m.syncChannels[*conn.Id] = storeSyncChannelConfig

    m.eventBus.GetChannelManager().CreateChannel(syncChannel)
    m.eventBus.GetChannelManager().MarkChannelAsGalactic(syncChannel, topicPrefix + syncChannel, conn)

    return nil
}

func (m *storeManager) OpenGalacticStore(name string, conn *bridge.Connection) (BusStore, error) {
    return m.OpenGalacticStoreWithItemType(name, conn, nil)
}

func (m *storeManager) OpenGalacticStoreWithItemType(
        name string, conn *bridge.Connection, itemType reflect.Type) (BusStore, error) {

    m.syncChannelsLock.RLock()
    chanConf, ok := m.syncChannels[*conn.Id]
    m.syncChannelsLock.RUnlock()

    if !ok {
        return nil, fmt.Errorf("sync channel is not configured for this connection")
    }

    m.storesLock.Lock()
    defer m.storesLock.Unlock()

    store, ok := m.stores[name]

    if ok {
        if store.IsGalactic() {
            return store, nil
        } else {
            return store, fmt.Errorf("there is a local store with the same name")
        }
    }

    m.stores[name] = newBusStore(name, m.eventBus, &galacticStoreConfig{
        syncChannelConfig: chanConf,
        itemType: itemType,
    })
    return m.stores[name], nil
}
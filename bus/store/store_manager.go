// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package store

import "sync"

// StoreManager interface controls all access to BusStores
type StoreManager interface {
    // Create a new Store, if the store already exists, then it will be returned.
    CreateStore(name string) BusStore
    // Get a reference to the existing store. Returns nil if the store doesn't exist.
    GetStore(name string) BusStore
    // Deletes a store.
    DestroyStore(name string) bool
}

type storeManager struct {
    stores          map[string]BusStore
    storesLock      sync.RWMutex
}

func NewStoreManager() StoreManager {
    m := new(storeManager)
    m.stores = make(map[string] BusStore)
    return m
}

func (m *storeManager) CreateStore(name string) BusStore {
    m.storesLock.Lock()
    defer m.storesLock.Unlock()

    store, ok := m.stores[name]

    if ok {
        return store
    }

    m.stores[name] = newBusStore(name)
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

    _, ok := m.stores[name]
    if ok {
        delete(m.stores, name)
    }
    return ok
}

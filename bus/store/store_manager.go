package store

import "sync"

// StoreManager interface controls all access to BusStores
type StoreManager interface {
    CreateStore(name string) BusStore
    GetStore(name string) BusStore
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

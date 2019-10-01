package store

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "sync"
    "sync/atomic"
)

func createManager() StoreManager {
    return NewStoreManager()
}

func TestStoreManager_CreateStore(t *testing.T) {
    storeManager := createManager()
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
    storeManager := createManager()
    storeManager.CreateStore("testStore")
    store := storeManager.GetStore("testStore")
    assert.Equal(t, store.GetName(), "testStore")

    assert.Nil(t, storeManager.GetStore("invalid-store"))
}

func TestStoreManager_DestroyStore(t *testing.T) {
    storeManager := createManager()
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


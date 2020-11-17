// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
    "fmt"
    "sync"
)

type StoreChangeHandlerFunction func(change *StoreChange)

// Interface for subscribing for store changes
type StoreStream interface {
    // Subscribe to the store changes stream.
    Subscribe(handler StoreChangeHandlerFunction) error
    // Unsubscribe from the stream.
    Unsubscribe() error
}

type streamFilter struct {
    states        []interface{}
    itemId        string
    matchAllItems bool
}

func (f *streamFilter) match(change *StoreChange) bool {
    if f.matchAllItems || f.itemId == change.Id {
        if len(f.states) == 0 {
            return true
        }

        for _, s := range f.states {
            if s == change.State {
                return true
            }
        }

    }
    return false
}

type storeStream struct {
    handler  StoreChangeHandlerFunction
    lock     sync.RWMutex
    store    *busStore
    filter   *streamFilter
}

func newStoreStream(store *busStore, filter *streamFilter) *storeStream {
    stream := new(storeStream)
    stream.store = store
    stream.filter = filter
    return stream
}

func (s *storeStream) Subscribe(handler StoreChangeHandlerFunction) error {
    if handler == nil {
        return fmt.Errorf("invalid StoreChangeHandlerFunction")
    }

    s.lock.Lock()
    if s.handler != nil {
        s.lock.Unlock()
        return fmt.Errorf("stream already subscribed")
    }
    s.handler = handler
    s.lock.Unlock()

    s.store.onStreamSubscribe(s)
    return nil
}

func (s *storeStream) Unsubscribe() error {
    s.lock.Lock()
    if s.handler == nil {
        s.lock.Unlock()
        return fmt.Errorf("stream not subscribed")
    }
    s.handler = nil
    s.lock.Unlock()

    s.store.onStreamUnsubscribe(s)
    return nil
}

func (s *storeStream) onStoreChange(change *StoreChange) {
    if !s.filter.match(change) {
        return
    }

    s.lock.RLock()
    defer s.lock.RUnlock()
    if s.handler != nil {
        go s.handler(change)
    }
}

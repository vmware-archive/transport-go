// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
    "sync"
    "fmt"
)

type MutationRequest struct {
    Request        interface{}
    RequestType    interface{}
    SuccessHandler func(interface{})
    ErrorHandler   func(interface{})
}

type MutationRequestHandlerFunction func(mutationReq *MutationRequest)

// Interface for subscribing for mutation requests
type MutationStoreStream interface {
    // Subscribe to the mutation requests stream.
    Subscribe(handler MutationRequestHandlerFunction) error
    // Unsubscribe from the stream.
    Unsubscribe() error
}

type mutationStreamFilter struct {
    requestTypes      []interface{}
}

func (f *mutationStreamFilter) match(mutationReq *MutationRequest) bool {
    if len(f.requestTypes) == 0 {
        return true
    }

    for _, s := range f.requestTypes {
        if mutationReq.RequestType == s {
            return true
        }
    }

    return false
}

type mutationStoreStream struct {
    handler MutationRequestHandlerFunction
    lock    sync.RWMutex
    store   *busStore
    filter  *mutationStreamFilter
}

func newMutationStoreStream(store *busStore, filter *mutationStreamFilter) *mutationStoreStream {
    stream := new(mutationStoreStream)
    stream.store = store
    stream.filter = filter
    return stream
}

func (ms *mutationStoreStream) Subscribe(handler MutationRequestHandlerFunction) error {
    if handler == nil {
        return fmt.Errorf("invalid MutationRequestHandlerFunction")
    }

    ms.lock.Lock()
    if ms.handler != nil {
        ms.lock.Unlock()
        return fmt.Errorf("stream already subscribed")
    }
    ms.handler = handler
    ms.lock.Unlock()

    ms.store.onMutationStreamSubscribe(ms)
    return nil
}

func (ms *mutationStoreStream) Unsubscribe() error {
    ms.lock.Lock()
    if ms.handler == nil {
        ms.lock.Unlock()
        return fmt.Errorf("stream not subscribed")
    }
    ms.handler = nil
    ms.lock.Unlock()

    ms.store.onMutationStreamUnsubscribe(ms)
    return nil
}

func (ms *mutationStoreStream) onMutationRequest(mutationReq *MutationRequest) {
    if !ms.filter.match(mutationReq) {
        return
    }

    ms.lock.RLock()
    defer ms.lock.RUnlock()
    if ms.handler != nil {
        go ms.handler(mutationReq)
    }
}

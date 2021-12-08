// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vmware/transport-go/model"
	"sync"
)

type transactionType int

const (
	asyncTransaction transactionType = iota
	syncTransaction
)

type BusTransactionReadyFunction func(responses []*model.Message)

type BusTransaction interface {
	// Sends a request to a channel as a part of this transaction.
	SendRequest(channel string, payload interface{}) error
	//  Wait for a store to be initialized as a part of this transaction.
	WaitForStoreReady(storeName string) error
	// Registers a new complete handler. Once all responses to requests have been received,
	// the transaction is complete.
	OnComplete(completeHandler BusTransactionReadyFunction) error
	// Register a new error handler. If an error is thrown by any of the responders, the transaction
	// is aborted and the error sent to the registered errorHandlers.
	OnError(errorHandler MessageErrorFunction) error
	// Commit the transaction, all requests will be sent and will wait for responses.
	// Once all the responses are in, onComplete handlers will be called with the responses.
	Commit() error
}

type transactionState int

const (
	uncommittedState transactionState = iota
	committedState
	completedState
	abortedState
)

type busTransactionRequest struct {
	requestIndex int
	storeName    string
	channelName  string
	payload      interface{}
}

type busTransaction struct {
	transactionType    transactionType
	state              transactionState
	lock               sync.Mutex
	requests           []*busTransactionRequest
	responses          []*model.Message
	onCompleteHandlers []BusTransactionReadyFunction
	onErrorHandlers    []MessageErrorFunction
	bus                EventBus
	completedRequests  int
}

func newBusTransaction(bus EventBus, transactionType transactionType) BusTransaction {
	transaction := new(busTransaction)

	transaction.bus = bus
	transaction.state = uncommittedState
	transaction.transactionType = transactionType
	transaction.requests = make([]*busTransactionRequest, 0)
	transaction.onCompleteHandlers = make([]BusTransactionReadyFunction, 0)
	transaction.onErrorHandlers = make([]MessageErrorFunction, 0)
	transaction.completedRequests = 0

	return transaction
}

func (tr *busTransaction) checkUncommittedState() error {
	if tr.state != uncommittedState {
		return fmt.Errorf("transaction has already been committed")
	}
	return nil
}

func (tr *busTransaction) SendRequest(channel string, payload interface{}) error {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	if err := tr.checkUncommittedState(); err != nil {
		return err
	}

	tr.requests = append(tr.requests, &busTransactionRequest{
		channelName:  channel,
		payload:      payload,
		requestIndex: len(tr.requests),
	})

	return nil
}

func (tr *busTransaction) WaitForStoreReady(storeName string) error {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	if err := tr.checkUncommittedState(); err != nil {
		return err
	}

	if tr.bus.GetStoreManager().GetStore(storeName) == nil {
		return fmt.Errorf("cannot find store '%s'", storeName)
	}

	tr.requests = append(tr.requests, &busTransactionRequest{
		storeName:    storeName,
		requestIndex: len(tr.requests),
	})

	return nil
}

func (tr *busTransaction) OnComplete(completeHandler BusTransactionReadyFunction) error {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	if err := tr.checkUncommittedState(); err != nil {
		return err
	}

	tr.onCompleteHandlers = append(tr.onCompleteHandlers, completeHandler)
	return nil
}

func (tr *busTransaction) OnError(errorHandler MessageErrorFunction) error {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	if err := tr.checkUncommittedState(); err != nil {
		return err
	}

	tr.onErrorHandlers = append(tr.onErrorHandlers, errorHandler)
	return nil
}

func (tr *busTransaction) Commit() error {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	if err := tr.checkUncommittedState(); err != nil {
		return err
	}

	if len(tr.requests) == 0 {
		return fmt.Errorf("cannot commit empty transaction")
	}

	tr.state = committedState

	// init responses slice
	tr.responses = make([]*model.Message, len(tr.requests))

	if tr.transactionType == asyncTransaction {
		tr.startAsyncTransaction()
	} else {
		tr.startSyncTransaction()
	}

	return nil
}

func (tr *busTransaction) startSyncTransaction() {
	tr.executeRequest(tr.requests[0])
}

func (tr *busTransaction) executeRequest(request *busTransactionRequest) {
	if request.storeName != "" {
		tr.waitForStore(request)
	} else {
		tr.sendRequest(request)
	}
}

func (tr *busTransaction) startAsyncTransaction() {
	for _, req := range tr.requests {
		tr.executeRequest(req)
	}
}

func (tr *busTransaction) sendRequest(req *busTransactionRequest) {
	reqId := uuid.New()

	mh, err := tr.bus.ListenOnceForDestination(req.channelName, &reqId)
	if err != nil {
		tr.onTransactionError(err)
		return
	}

	mh.Handle(func(message *model.Message) {
		tr.onTransactionRequestSuccess(req, message)
	}, func(e error) {
		tr.onTransactionError(e)
	})

	tr.bus.SendRequestMessage(req.channelName, req.payload, &reqId)
}

func (tr *busTransaction) onTransactionError(err error) {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	if tr.state == abortedState {
		return
	}

	tr.state = abortedState
	for _, errorHandler := range tr.onErrorHandlers {
		go errorHandler(err)
	}
}

func (tr *busTransaction) waitForStore(req *busTransactionRequest) {
	store := tr.bus.GetStoreManager().GetStore(req.storeName)
	if store == nil {
		tr.onTransactionError(fmt.Errorf("cannot find store '%s'", req.storeName))
		return
	}
	store.WhenReady(func() {
		tr.onTransactionRequestSuccess(req, &model.Message{
			Direction: model.ResponseDir,
			Payload:   store.AllValuesAsMap(),
		})
	})
}

func (tr *busTransaction) onTransactionRequestSuccess(req *busTransactionRequest, message *model.Message) {
	var triggerOnCompleteHandler = false
	tr.lock.Lock()

	if tr.state == abortedState {
		tr.lock.Unlock()
		return
	}

	tr.responses[req.requestIndex] = message
	tr.completedRequests++

	if tr.completedRequests == len(tr.requests) {
		tr.state = completedState
		triggerOnCompleteHandler = true
	}

	tr.lock.Unlock()

	if triggerOnCompleteHandler {
		for _, completeHandler := range tr.onCompleteHandlers {
			go completeHandler(tr.responses)
		}
		return
	}

	// If this is a sync transaction execute the next request
	if tr.transactionType == syncTransaction && req.requestIndex < len(tr.requests)-1 {
		tr.executeRequest(tr.requests[req.requestIndex+1])
	}
}

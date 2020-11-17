// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bus

import (
    "errors"
    "github.com/stretchr/testify/assert"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
    "sync"
    "sync/atomic"
    "testing"
)

func TestBusTransaction_OnCompleteSync(t *testing.T) {

    bus := newTestEventBus()

    bus.GetChannelManager().CreateChannel("test-channel")

    var channelReqMessage *model.Message
    var requestCounter = 0

    wg := sync.WaitGroup{}

    mh,_ := bus.ListenRequestStream("test-channel")
    mh.Handle(func(message *model.Message) {
        requestCounter++
        channelReqMessage = message
        wg.Done()
    }, func(e error) {
        assert.Fail(t, "unexpected error")
    })

    tr := newBusTransaction(bus, syncTransaction)

    bus.GetStoreManager().CreateStore("testStore")
    assert.Nil(t, tr.WaitForStoreReady("testStore"))
    assert.Nil(t, tr.SendRequest("test-channel", "sample-request"))

    var completeCounter int64

    tr.OnComplete(func(responses []*model.Message) {
        atomic.AddInt64(&completeCounter, 1)
        wg.Done()
    })

    tr.OnError(func(e error) {
        assert.Fail(t, "unexpected error")
    })

    tr.OnComplete(func(responses []*model.Message) {
        atomic.AddInt64(&completeCounter, 1)
        assert.Equal(t, len(responses), 2)
        assert.Equal(t, responses[1].Channel, "test-channel")
        assert.Equal(t, responses[1].Payload, "sample-response")
        wg.Done()
    })

    assert.Equal(t, requestCounter, 0)

    wg.Add(1)

    assert.Nil(t, tr.Commit())

    go bus.GetStoreManager().CreateStore("testStore").Initialize()

    wg.Wait()

    assert.Equal(t, requestCounter, 1)
    assert.NotNil(t, channelReqMessage)

    assert.Equal(t, channelReqMessage.Payload, "sample-request")

    for i := 0; i < 50; i++ {
        bus.SendResponseMessage("test-channel", "general-message", nil)
    }

    assert.Equal(t, completeCounter, int64(0))

    wg.Add(2)
    bus.SendResponseMessage("test-channel", "sample-response", channelReqMessage.DestinationId)

    wg.Wait()

    assert.Equal(t, tr.(*busTransaction).state, completedState)

    assert.Equal(t, completeCounter, int64(2))

    bus.SendResponseMessage("test-channel", "sample-response2", channelReqMessage.DestinationId)
    assert.Equal(t, completeCounter, int64(2))
}

func TestBusTransaction_OnCompleteErrorHandling(t *testing.T) {

    bus := newTestEventBus()

    tr := newBusTransaction(bus, syncTransaction)

    assert.EqualError(t,  tr.Commit(), "cannot commit empty transaction")

    assert.Equal(t, tr.(*busTransaction).state, uncommittedState)

    bus.GetStoreManager().CreateStore("testStore")
    assert.Nil(t, tr.WaitForStoreReady("testStore"))

    assert.EqualError(t, tr.WaitForStoreReady("invalid-store"), "cannot find store 'invalid-store'")

    tr.Commit()

    assert.EqualError(t, tr.OnComplete(func(responses []*model.Message) {}), "transaction has already been committed")

    assert.Equal(t, tr.(*busTransaction).state, committedState)
    assert.EqualError(t,  tr.Commit(), "transaction has already been committed")

    assert.EqualError(t,  tr.WaitForStoreReady("test"), "transaction has already been committed")
    assert.EqualError(t,  tr.SendRequest("test", "test"), "transaction has already been committed")
}

func TestBusTransaction_OnErrorSync(t *testing.T) {

    bus := newTestEventBus()

    tr := newBusTransaction(bus, syncTransaction)

    bus.GetStoreManager().CreateStore("testStore")
    assert.Nil(t, tr.WaitForStoreReady("testStore"))

    bus.GetChannelManager().CreateChannel("test-channel")

    var channelReqMessage *model.Message
    var requestCounter = 0

    wg := sync.WaitGroup{}

    mh,_ := bus.ListenRequestStream("test-channel")
    mh.Handle(func(message *model.Message) {
        requestCounter++
        channelReqMessage = message
        wg.Done()
    }, func(e error) {
    })

    tr.SendRequest("test-channel", "sample-request")
    tr.SendRequest("test-channel", "sample-request")
    tr.SendRequest("test-channel", "sample-request")


    tr.OnComplete(func(responses []*model.Message) {
        assert.Fail(t, "invalid state")
    })

    var errorHandlerCount int64 = 0
    tr.OnError(func(e error) {
        atomic.AddInt64(&errorHandlerCount, 1)
        wg.Done()
    })

    tr.OnError(func(e error) {
        atomic.AddInt64(&errorHandlerCount, 1)
        assert.EqualError(t, e, "test-error")
        wg.Done()
    })

    tr.Commit()

    assert.Equal(t, tr.(*busTransaction).state, committedState)

    wg.Add(1)

    bus.GetStoreManager().GetStore("testStore").Initialize()

    wg.Wait()

    assert.Equal(t, requestCounter, 1)
    assert.NotNil(t, channelReqMessage)

    wg.Add(2)
    bus.SendErrorMessage("test-channel", errors.New("test-error"), channelReqMessage.DestinationId)

    wg.Wait()

    assert.Equal(t, tr.(*busTransaction).state, abortedState)

    assert.Equal(t, requestCounter, 1)
    assert.Equal(t, errorHandlerCount, int64(2))

    assert.EqualError(t,  tr.Commit(), "transaction has already been committed")
}

func TestBusTransaction_OnCompleteAsync(t *testing.T) {

    bus := newTestEventBus()

    bus.GetChannelManager().CreateChannel("test-channel")

    var channelReqMessage *model.Message
    var requestCounter = 0

    wg := sync.WaitGroup{}

    mh,_ := bus.ListenRequestStream("test-channel")
    mh.Handle(func(message *model.Message) {
        requestCounter++
        channelReqMessage = message
        wg.Done()
    }, func(e error) {
        assert.Fail(t, "unexpected error")
    })

    tr := newBusTransaction(bus, asyncTransaction)

    bus.GetStoreManager().CreateStore("testStore")
    assert.Nil(t, tr.WaitForStoreReady("testStore"))
    assert.Nil(t, tr.WaitForStoreReady("testStore"))
    bus.GetStoreManager().CreateStore("testStore2")
    assert.Nil(t, tr.WaitForStoreReady("testStore2"))
    bus.GetStoreManager().CreateStore("testStore3")
    assert.Nil(t, tr.WaitForStoreReady("testStore3"))
    assert.Nil(t, tr.SendRequest("test-channel", "sample-request"))

    var completeCounter int64

    tr.OnComplete(func(responses []*model.Message) {
        atomic.AddInt64(&completeCounter, 1)
        wg.Done()
    })

    tr.OnComplete(func(responses []*model.Message) {
        atomic.AddInt64(&completeCounter, 1)
        assert.Equal(t, len(responses), 5)
        assert.Equal(t, responses[4].Channel, "test-channel")
        assert.Equal(t, responses[4].Payload, "sample-response")
        wg.Done()
    })

    wg.Add(1)
    assert.Nil(t, tr.Commit())
    wg.Wait()

    assert.NotNil(t, bus.GetStoreManager().GetStore("testStore"))
    assert.NotNil(t, bus.GetStoreManager().GetStore("testStore2"))
    assert.NotNil(t, bus.GetStoreManager().GetStore("testStore3"))
    assert.Equal(t, requestCounter, 1)
    assert.NotNil(t, channelReqMessage)
    assert.Equal(t, channelReqMessage.Payload, "sample-request")

    for i := 0; i < 20; i++ {
        bus.SendResponseMessage("test-channel", "general-message", nil)
    }

    assert.Equal(t, completeCounter, int64(0))

    wg.Add(2)

    bus.SendResponseMessage("test-channel", "sample-response", channelReqMessage.DestinationId)
    bus.GetStoreManager().GetStore("testStore").Initialize()
    bus.GetStoreManager().GetStore("testStore2").Initialize()
    bus.GetStoreManager().GetStore("testStore3").Initialize()

    wg.Wait()

    assert.Equal(t, completeCounter, int64(2))
}

func TestBusTransaction_OnErrorAsync(t *testing.T) {

    bus := newTestEventBus()

    tr := newBusTransaction(bus, asyncTransaction)

    bus.GetChannelManager().CreateChannel("test-channel")
    bus.GetChannelManager().CreateChannel("test-channel2")

    var channelReqMessage, channelReqMessage2 *model.Message


    wg := sync.WaitGroup{}

    mh,_ := bus.ListenRequestStream("test-channel")
    mh.Handle(func(message *model.Message) {
        channelReqMessage = message
        wg.Done()
    }, func(e error) {
    })

    mh2,_ := bus.ListenRequestStream("test-channel2")
    mh2.Handle(func(message *model.Message) {
        channelReqMessage2 = message
        wg.Done()
    }, func(e error) {
    })


    tr.OnComplete(func(responses []*model.Message) {
        assert.Fail(t, "invalid state")
    })

    var errorHandlerCount int64 = 0
    tr.OnError(func(e error) {
        atomic.AddInt64(&errorHandlerCount, 1)
        assert.EqualError(t, e, "test-error")
        wg.Done()
    })

    tr.SendRequest("test-channel", "sample-request")
    tr.SendRequest("test-channel2", "sample-request2")

    wg.Add(2)
    tr.Commit()
    wg.Wait()

    wg.Add(1)
    bus.SendErrorMessage("test-channel2", errors.New("test-error"), channelReqMessage2.DestinationId)

    wg.Wait()

    assert.Equal(t, errorHandlerCount, int64(1))

    for i := 0; i < 50; i++ {
        bus.SendErrorMessage("test-channel", errors.New("test-error-2"), channelReqMessage.DestinationId)
    }

    assert.Equal(t, errorHandlerCount, int64(1))
}

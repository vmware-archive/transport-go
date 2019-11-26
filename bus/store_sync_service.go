// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package bus

import (
    "go-bifrost/model"
    "strings"
    "sync"
    "github.com/google/uuid"
)

const (
    openStoreRequest = "openStore"
    updateStoreRequest = "updateStore"
    closeStoreRequest = "closeStore"
    galacticStoreSyncUpdate = "galacticStoreSyncUpdate"
    galacticStoreSyncRemove = "galacticStoreSyncRemove"
)

type storeSyncService struct {
    bus                EventBus
    lock               sync.Mutex
    syncClients        map[string]*syncClientChannel
    syncStoreListeners map[string]*syncStoreListener
}

type syncStoreListener struct {
    storeStream        StoreStream
    clientSyncChannels map[string]bool
    lock               sync.RWMutex
}


type syncClientChannel struct {
    channelName           string
    clientRequestListener MessageHandler
    openStores            map[string]bool
}

func newStoreSyncService(bus EventBus) *storeSyncService {
    syncService := &storeSyncService{
        bus:                bus,
        syncClients:        make(map[string]*syncClientChannel),
        syncStoreListeners: make(map[string]*syncStoreListener),
    }
    syncService.init()
    return syncService
}

func (syncService *storeSyncService) init() {
    syncService.bus.AddMonitorEventListener(
        func(monitorEvt *MonitorEvent) {
            if !strings.HasPrefix(monitorEvt.EntityName, "fabric-store-sync.") {
                // not a store sync channel, ignore the message
                return
            }

            switch monitorEvt.EventType {
            case FabricEndpointSubscribeEvt:
                syncService.openNewClientSyncChannel(monitorEvt.EntityName)
            case ChannelDestroyedEvt:
                syncService.closeClientSyncChannel(monitorEvt.EntityName)
            }
        },
        FabricEndpointSubscribeEvt, ChannelDestroyedEvt)
}

func (syncService *storeSyncService) openNewClientSyncChannel(channelName string) {
    syncService.lock.Lock()
    defer syncService.lock.Unlock()

    if _, ok := syncService.syncClients[channelName]; ok {
        // channel already opened.
        return
    }

    syncClient := &syncClientChannel{
        channelName: channelName,
        openStores: make(map[string]bool),
    }
    syncClient.clientRequestListener, _ = syncService.bus.ListenRequestStream(channelName)
    if syncClient.clientRequestListener != nil {
        syncClient.clientRequestListener.Handle(
            func(message *model.Message) {
                request, reqOk := message.Payload.(*model.Request)
                if !reqOk || request.Payload == nil {
                    return
                }
                var storeRequest map[string]interface{}
                storeRequest, ok := request.Payload.(map[string]interface{})
                if !ok {
                    return
                }

                switch request.Request {
                case openStoreRequest:
                    syncService.openStore(syncClient, storeRequest, request.Id)
                case closeStoreRequest:
                    syncService.closeStore(syncClient, storeRequest, request.Id)
                case updateStoreRequest:
                    syncService.updateStore(syncClient, storeRequest, request.Id)
                }
            }, func(e error) {})
    }
    syncService.syncClients[channelName] = syncClient
}

func (syncService *storeSyncService) closeClientSyncChannel(channelName string) {
    syncService.lock.Lock()
    defer syncService.lock.Unlock()

    syncClient, ok := syncService.syncClients[channelName]
    if !ok || syncClient == nil {
        // client is already closed
        return
    }

    for storeId := range syncClient.openStores {
        listener := syncService.syncStoreListeners[storeId]
        if listener != nil {
            listener.removeChannel(channelName)
            if listener.isEmpty() {
                listener.unsubscribe()
                delete(syncService.syncStoreListeners, storeId)
            }
        }
    }

    delete(syncService.syncClients, channelName)
}

func (syncService *storeSyncService) openStore(
        syncClient *syncClientChannel, request map[string]interface{}, reqId *uuid.UUID) {

    storeId, ok := getStingProperty("storeId", request)
    if !ok || storeId == "" {
        syncService.sendErrorResponse(syncClient.channelName, "Invalid OpenStoreRequest", reqId)
        return
    }

    store := syncService.bus.GetStoreManager().GetStore(storeId)
    if store == nil {
        syncService.sendErrorResponse(
                syncClient.channelName, "Cannot open non-existing store: " + storeId, reqId)
        return
    }

    syncService.lock.Lock()
    defer syncService.lock.Unlock()

    syncClient.openStores[storeId] = true

    storeListener, ok := syncService.syncStoreListeners[storeId]
    if !ok {
        storeListener = newSyncStoreListener(syncService.bus, store)
        syncService.syncStoreListeners[storeId] = storeListener
    }
    storeListener.addChannel(syncClient.channelName)

    store.WhenReady(func() {
        items, version :=  store.AllValuesAndVersion()

        syncService.bus.SendResponseMessage(syncClient.channelName,
                model.NewStoreContentResponse(storeId, items, version), nil)
    })
}

func (syncService *storeSyncService) closeStore(
        syncClient *syncClientChannel, request map[string]interface{}, reqId *uuid.UUID) {

    storeId, ok := getStingProperty("storeId", request)
    if !ok || storeId == "" {
        syncService.sendErrorResponse(syncClient.channelName, "Invalid CloseStoreRequest", reqId)
        return
    }

    syncService.lock.Lock()
    defer syncService.lock.Unlock()

    delete(syncClient.openStores, storeId)

    storeListener, ok := syncService.syncStoreListeners[storeId]
    if ok && storeListener != nil {
        storeListener.removeChannel(syncClient.channelName)
        if storeListener.isEmpty() {
            storeListener.unsubscribe()
            delete(syncService.syncStoreListeners, storeId)
        }
    }
}

func (syncService *storeSyncService) updateStore(
        syncClient *syncClientChannel, request map[string]interface{}, reqId *uuid.UUID) {

    storeId, ok := getStingProperty("storeId", request)
    if !ok || storeId == "" {
        syncService.sendErrorResponse(
                syncClient.channelName, "Invalid UpdateStoreRequest: missing storeId", reqId)
        return
    }
    itemId, ok := getStingProperty("itemId", request)
    if !ok || itemId == "" {
        syncService.sendErrorResponse(
                syncClient.channelName, "Invalid UpdateStoreRequest: missing itemId", reqId)
        return
    }

    store := syncService.bus.GetStoreManager().GetStore(storeId)
    if store == nil {
        syncService.sendErrorResponse(
                syncClient.channelName, "Cannot update non-existing store: " + storeId, reqId)
        return
    }

    rawValue, ok := request["newItemValue"]
    if rawValue == nil {
        store.Remove(itemId, galacticStoreSyncRemove)
    } else {
        deserializedValue, err := deserializeRawValue(store.GetItemType(), rawValue)
        if err != nil || deserializedValue == nil {
            errMsg :=  "Cannot deserialize UpdateStoreRequest item value"
            if err != nil {
                errMsg = "Cannot deserialize UpdateStoreRequest item value: " + err.Error()
            }
            syncService.sendErrorResponse(syncClient.channelName, errMsg, reqId)
            return
        }
        store.Put(itemId, deserializedValue, galacticStoreSyncUpdate)
    }
}

func getStingProperty(id string, request map[string]interface{}) (string, bool) {
    propValue, ok := request[id]
    if !ok || propValue == nil {
        return "", false
    }
    stringValue, ok := propValue.(string)
    return stringValue, ok
}

func (syncService *storeSyncService) sendErrorResponse(
        clientChannel string, errorMsg string, reqId *uuid.UUID) {

    syncService.bus.SendResponseMessage(clientChannel, &model.Response{
        Id: reqId,
        Error: true,
        ErrorCode: 1,
        ErrorMessage: errorMsg,
    }, nil)
}

func newSyncStoreListener(bus EventBus, store BusStore) *syncStoreListener {

    listener := &syncStoreListener{
        storeStream: store.OnAllChanges(),
        clientSyncChannels: make(map[string]bool),
    }

    listener.storeStream.Subscribe(func(change *StoreChange) {
        updateStoreResp := model.NewUpdateStoreResponse(
                store.GetName(), change.Id, change.Value, change.StoreVersion)
        if change.IsDeleteChange {
            updateStoreResp.NewItemValue = nil
        }

        listener.lock.RLock()
        defer listener.lock.RUnlock()

        for chName := range listener.clientSyncChannels {
            bus.SendResponseMessage(chName, updateStoreResp, nil)
        }
    })

    return listener
}

func (l *syncStoreListener) unsubscribe() {

    l.storeStream.Unsubscribe()
}

func (l *syncStoreListener) addChannel(clientChannel string) {
    l.lock.Lock()
    defer l.lock.Unlock()
    l.clientSyncChannels[clientChannel] = true
}

func (l *syncStoreListener) removeChannel(clientChannel string) {
    l.lock.Lock()
    defer l.lock.Unlock()
    delete(l.clientSyncChannels, clientChannel)
}

func (l *syncStoreListener) isEmpty() bool {
    l.lock.Lock()
    defer l.lock.Unlock()
    return len(l.clientSyncChannels) == 0
}

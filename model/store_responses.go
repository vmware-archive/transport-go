// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package model

type StoreContentResponse struct {
    Items        map[string]interface{} `json:"items"`
    ResponseType string                 `json:"responseType"` // should be "storeContentResponse"
    StoreId      string                 `json:"storeId"`
    StoreVersion int64                    `json:"storeVersion"`
}

func NewStoreContentResponse(
        storeId string, items map[string]interface{}, storeVersion int64) *StoreContentResponse {

    return &StoreContentResponse{
        ResponseType: "storeContentResponse",
        StoreId: storeId,
        Items: items,
        StoreVersion: storeVersion,
    }
}

type UpdateStoreResponse struct {
    ItemId       string      `json:"itemId"`
    NewItemValue interface{} `json:"newItemValue"`
    ResponseType string      `json:"responseType"` // should be "updateStoreResponse"
    StoreId      string      `json:"storeId"`
    StoreVersion int64         `json:"storeVersion"`
}

func NewUpdateStoreResponse(
        storeId string, itemId string, newValue interface{},storeVersion int64) *UpdateStoreResponse {

    return &UpdateStoreResponse{
        ResponseType: "updateStoreResponse",
        StoreId: storeId,
        StoreVersion: storeVersion,
        ItemId: itemId,
        NewItemValue: newValue,
    }
}
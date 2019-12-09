// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package model

import (
    "encoding/json"
    "reflect"
)

func ConvertValueToType(value interface{}, targetType reflect.Type) (interface{}, error) {
    if targetType == nil {
        return value, nil
    }

    itemType := targetType
    var isTargetTypePointer bool

    if itemType.Kind() == reflect.Ptr {
        isTargetTypePointer = true
        itemType = itemType.Elem()
    }

    decodedValuePtr := reflect.New(itemType).Interface()

    marshaledValue, _ := json.Marshal(value)
    decodeErr := json.Unmarshal(marshaledValue, decodedValuePtr)

    if decodeErr != nil {
        return  nil, decodeErr
    }

    if isTargetTypePointer {
        return decodedValuePtr, nil
    } else {
        return reflect.ValueOf(decodedValuePtr).Elem().Interface(), nil
    }
}

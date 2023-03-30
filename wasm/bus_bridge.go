//go:build js && wasm
// +build js,wasm

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package wasm

import (
	"reflect"
	"syscall/js"

	"github.com/google/uuid"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
)

type TransportWasmBridge struct {
	busRef bus.EventBus
}

func NewTransportWasmBridge(busRef bus.EventBus) *TransportWasmBridge {
	return &TransportWasmBridge{
		busRef: busRef,
	}
}

func (t *TransportWasmBridge) sendRequestMessageWrapper() js.Func {
	var jsFunc js.Func
	jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var destId *uuid.UUID
		channel := args[0].String()
		payload := args[1].JSValue()
		if len(args) > 2 {
			dId, err := uuid.Parse(args[2].String())
			if err != nil {
				errCtor := js.Global().Get("Error")
				js.Global().Get("console").Call("error", errCtor.New(err.Error()))
				return nil
			}
			destId = &dId
		}
		t.busRef.SendRequestMessage(channel, payload, destId)
		return nil
	})
	return jsFunc
}

func (t *TransportWasmBridge) sendResponseMessageWrapper() js.Func {
	var jsFunc js.Func
	jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var destId *uuid.UUID
		channel := args[0].String()
		payload := args[1].JSValue()
		if len(args) > 2 {
			dId, err := uuid.Parse(args[2].String())
			if err != nil {
				errCtor := js.Global().Get("Error")
				js.Global().Get("console").Call("error", errCtor.New(err.Error()))
				return nil
			}
			destId = &dId
		}
		t.busRef.SendResponseMessage(channel, payload, destId)
		return nil
	})
	return jsFunc
}

func (t *TransportWasmBridge) listenStreamWrapper() js.Func {
	var jsFunc js.Func
	jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler, err := t.busRef.ListenStream(args[0].String())
		if err != nil {
			errCtor := js.Global().Get("Error")
			js.Global().Get("console").Call("error", errCtor.New(err.Error()))
			return nil
		}

		jsFuncRefs := make([]js.Func, 0)
		closerFuncRef := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			handler.Close()

			// release the resources allocated for js.Func when closing the subscription
			for _, ref := range jsFuncRefs {
				ref.Release()
			}

			return nil
		})

		// store js.Func handles for resource cleanup later
		jsFuncRefs = append(jsFuncRefs, getResponseHandlerJsCallbackFunc(handler), closerFuncRef)

		return map[string]interface{}{
			"handle": jsFuncRefs[0],
			"close":  jsFuncRefs[1],
		}
	})
	return jsFunc
}

func (t *TransportWasmBridge) listenStreamWithIdWrapper() js.Func {
	var jsFunc js.Func
	jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) <= 1 {
			errCtor := js.Global().Get("Error")
			js.Global().Get("console").Call("error", errCtor.New("cannot listen to channel. missing destination ID"))
			return nil
		}

		destId, err := uuid.Parse(args[1].String())
		if err != nil {
			errCtor := js.Global().Get("Error")
			return errCtor.New(err.Error())
		}

		handler, err := t.busRef.ListenStreamForDestination(args[0].String(), &destId)
		if err != nil {
			errCtor := js.Global().Get("Error")
			return errCtor.New(err.Error())
		}

		jsFuncRefs := make([]js.Func, 0)
		closerFuncRef := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			handler.Close()

			// release the resources allocated for js.Func when closing the subscription
			for _, ref := range jsFuncRefs {
				ref.Release()
			}

			return nil
		})

		// store js.Func handles for resource cleanup later
		jsFuncRefs = append(jsFuncRefs, getResponseHandlerJsCallbackFunc(handler), closerFuncRef)

		return map[string]interface{}{
			"handle": jsFuncRefs[0],
			"close":  jsFuncRefs[1],
		}
	})
	return jsFunc
}

func (t *TransportWasmBridge) browserBusWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		b := js.Global().Get("AppEventBus")
		if b.Type() == js.TypeUndefined || b.Type() == js.TypeNull {
			errCtor := js.Global().Get("Error")
			return errCtor.New("bus not initialized")
		}
		return b
	})
}

func (t *TransportWasmBridge) SetUpTransportWasmAPI() {
	js.Global().Set("goBusInterface", map[string]interface{}{
		"sendRequestMessage":        t.sendRequestMessageWrapper(),
		"sendRequestMessageWithId":  t.sendRequestMessageWrapper(),
		"sendResponseMessage":       t.sendResponseMessageWrapper(),
		"sendResponseMessageWithId": t.sendResponseMessageWrapper(),
		"listenStream":              t.listenStreamWrapper(),
		"listenStreamWithId":        t.listenStreamWithIdWrapper(),
		"getBrowserBus":             t.browserBusWrapper(),
	})

	js.Global().Get("console").Call("log", "Transport WASM bridge set up. APIs are exposed at window.goBusInterface")
}

func getResponseHandlerJsCallbackFunc(handler bus.MessageHandler) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		successCallback := args[0]
		hasFailureCallback := len(args) > 1

		if (successCallback.Type() != js.TypeFunction) || (hasFailureCallback && args[1].Type() != js.TypeFunction) {
			errCtor := js.Global().Get("Error")
			js.Global().Get("console").Call("error", errCtor.New("invalid argument. not a function."))
			handler.Close()
			return nil
		}

		handler.Handle(func(m *model.Message) {
			// workaround for js.ValueOf() panicking for slices of primitive types such as []int, []string, []bool, or
			// a map containing such containers for their keys. this is quite dirty but hopefully when generics arrive
			// we should be able to handle this situation more elegantly & hope go team improves js package.
			var payload = m.Payload
			switch payload.(type) {
			case []string, []int, []bool, []float64, map[string]interface{}:
				payload = toWasmSafePayload(payload)
			}
			successCallback.Invoke(payload)
		}, func(e error) {
			if hasFailureCallback {
				args[1].Invoke(e.Error())
			} else {
				errCtor := js.Global().Get("Error")
				js.Global().Get("console").Call("error", errCtor.New(e.Error()))
			}
		})
		return nil
	})
}

func toWasmSafePayload(payload interface{}) interface{} {
	refl := reflect.ValueOf(payload)
	if refl.Kind() == reflect.Map {
		for _, keyV := range refl.MapKeys() {
			valV := refl.MapIndex(keyV)
			refl.SetMapIndex(keyV, reflect.ValueOf(toWasmSafePayload(valV.Interface())))
		}
		return payload
	} else if refl.Kind() == reflect.Slice || refl.Kind() == reflect.Array {
		var converted = make([]interface{}, 0)
		for i := 0; i < refl.Len(); i++ {
			converted = append(converted, refl.Index(i).Interface())
		}
		return converted
	}

	// let js.ValueOf() handle the rest
	return payload
}

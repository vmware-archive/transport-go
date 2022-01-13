//go:build js && wasm

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package wasm

import (
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
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// release the resources allocated for js.Func once this closure goes out of scope
		defer jsFunc.Release()

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
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// release the resources allocated for js.Func once this closure goes out of scope
		defer jsFunc.Release()

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
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler, err := t.busRef.ListenStream(args[0].String())
		if err != nil {
			errCtor := js.Global().Get("Error")
			return errCtor.New(err.Error())
		}

		closerFuncRef := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			handler.Close()

			// release the resources allocated for js.Func when closing the subscription
			for _, ref := range jsFuncRefs {
				ref.Release()
			}

			return nil
		})

		// store js.Func handles for resource cleanup later
		jsFuncRefs := []js.Func{jsFunc, getResponseHandlerJsCallbackFunc(handler), closerFuncRef}

		return map[string]interface{}{
			"handle": jsFuncRefs[1],
			"close":  jsFuncRefs[2],
		}
	})
	return jsFunc
}

func (t *TransportWasmBridge) listenStreamWithIdWrapper() js.Func {
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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

		closerFuncRef := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			handler.Close()

			// release the resources allocated for js.Func when closing the subscription
			for _, ref := range jsFuncRefs {
				ref.Release()
			}

			return nil
		})

		// store js.Func handles for resource cleanup later
		jsFuncRefs := []js.Func{jsFunc, getResponseHandlerJsCallbackFunc(handler), closerFuncRef}

		return map[string]interface{}{
			"handle": jsFuncRefs[1],
			"close":  jsFuncRefs[2],
		}
	})
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
		handler.Handle(func(m *model.Message) {
			jsCallback := args[0]
			if jsCallback.Type() != js.TypeFunction {
				errCtor := js.Global().Get("Error")
				js.Global().Get("console").Call("error", errCtor.New("invalid argument. not a function."))
				handler.Close()
				return
			}
			jsCallback.Invoke(m.Payload)
		}, func(e error) {
			errCtor := js.Global().Get("Error")
			js.Global().Get("console").Call("error", errCtor.New(e.Error()))
			handler.Close()
		})
		return nil
	})
}

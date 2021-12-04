//go:build js && wasm

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"time"

	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/wasm"
)

func init() {
	b := bus.GetBus()
	cm := b.GetChannelManager()
	cm.CreateChannel("goBusChan")
	cm.CreateChannel("broadcast")

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				b.SendResponseMessage("goBusChan", "sending", b.GetId())
				b.SendResponseMessage("broadcast", "hello world", nil)
			}
		}
	}()
}

func main() {
	twa := wasm.NewTransportWasmBridge(bus.GetBus())
	twa.SetUpTransportWasmAPI()

	fmt.Println("go bus id: ", bus.GetBus().GetId())

	loop := make(chan struct{})
	<-loop
}

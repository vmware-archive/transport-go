//go:build js && wasm
// +build js,wasm

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/wasm"
	"syscall/js"
	"time"

	"github.com/vmware/transport-go/bus"
)

func init() {
	b := bus.GetBus()
	cm := b.GetChannelManager()
	cm.CreateChannel("goBusChan")
	cm.CreateChannel("broadcast")

	NewX509CertParser()

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

type X509CertParser struct {
	requestChannel  string
	responseChannel string
}

func NewX509CertParser() *X509CertParser {
	p := &X509CertParser{
		requestChannel:  "x509-cert-parser-req",
		responseChannel: "x509-cert-parser-resp",
	}

	// set up bus channels for receiving requests and sending responses.
	busInstance := bus.GetBus()
	cm := busInstance.GetChannelManager()
	cm.CreateChannel(p.requestChannel)
	cm.CreateChannel(p.responseChannel)

	// set up a channel handler to accept and process requests from JS.
	// once parsed or an error occurs, the response will be sent down the responseChannel.
	handler, _ := busInstance.ListenRequestStream(p.requestChannel)
	handler.Handle(func(message *model.Message) {
		jsVal := message.Payload.(js.Value)
		cert, err := p.parseFromJsValueString(jsVal)
		if err != nil {
			_ = busInstance.SendErrorMessage(p.responseChannel, err, nil)
			return
		}

		response := map[string]interface{}{
			"issuer":    cert.Issuer.String(),
			"dnsNames":  cert.DNSNames,
			"notBefore": cert.NotBefore.String(),
			"notAfter":  cert.NotAfter.String(),
			"isCA":      cert.IsCA,
			"version":   cert.Version,
		}
		// successful parse results are sent to the response channel for the success handler in the client
		// to be invoked with.
		_ = busInstance.SendResponseMessage(p.responseChannel, response, nil)
	}, func(err error) {
		// failed parse results will be sent to the same channel as well, but the error handler will be invoked instead
		// in the client.
		_ = busInstance.SendErrorMessage(p.responseChannel, err, nil)
	})

	return p
}

func (p *X509CertParser) parseFromJsValueString(input js.Value) (*x509.Certificate, error) {
	var b = []byte(input.String())
	var cert *x509.Certificate
	var err error

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("invalid pem format")
	}

	switch block.Type {
	case "CERTIFICATE":
		cert, err = x509.ParseCertificate(block.Bytes)
	}

	return cert, err
}

func main() {
	twa := wasm.NewTransportWasmBridge(bus.GetBus())
	twa.SetUpTransportWasmAPI()

	fmt.Println("go bus id: ", bus.GetBus().GetId())

	loop := make(chan struct{})
	<-loop
}

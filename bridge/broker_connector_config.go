// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
	"crypto/tls"
	"net/http"
	"time"
)

type WebSocketConfig struct {
	WSPath    string // if UseWS is true, set this to your websocket path (e.g. '/fabric')
	UseTLS    bool // use TLS encryption with WebSocket connection
	TLSConfig *tls.Config // TLS config for WebSocket connection
	CertFile  string // X509 certificate for TLS
	KeyFile   string // matching key file for the X509 certificate
}

// BrokerConnectorConfig is a configuration used when connecting to a message broker
type BrokerConnectorConfig struct {
	Username              string
	Password              string
	ServerAddr            string
	UseWS                 bool   // use WebSocket instead of TCP
	WebSocketConfig       *WebSocketConfig
	HostHeader            string
	HeartBeatOut          time.Duration
	HeartBeatIn           time.Duration
	STOMPHeader           map[string]string
	HttpHeader            http.Header
}

// LoadX509KeyPairFromFiles loads from paths to x509 cert and its matching key files and initializes
// the Certificates field of the TLS config instance with their contents, only if both Certificates is
// an empty slice and GetCertificate is nil
func (b *WebSocketConfig) LoadX509KeyPairFromFiles(certFile, keyFile string) error {
	var err error
	config := b.TLSConfig
	configHasCert := len(config.Certificates) > 0 || config.GetCertificate != nil
	if !configHasCert || certFile != "" || keyFile != "" {
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
	}
	return err
}
// Copyright 2019 VMware Inc.

package bridge

// BrokerConnectorConfig is a configuration used when connecting to a message broker
type BrokerConnectorConfig struct {
    Username        string
    Password        string
    ServerAddr      string
    WSPath          string  // if UseWS is true, set this to your websocket path (e.g. '/fabric')
    UseWS           bool    // use WebSocket instead of TCP
    HostHeader      string
}

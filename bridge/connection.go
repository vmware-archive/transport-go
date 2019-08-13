package bridge

import "github.com/go-stomp/stomp"

// abstraction for connection types.
type Connection struct {
    Conn   *stomp.Conn
    WsConn *BridgeWsClient
}

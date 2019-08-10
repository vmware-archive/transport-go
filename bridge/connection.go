package bridge

import "github.com/go-stomp/stomp"

// abstraction for connection types.
type Connection struct {
    conn    *stomp.Conn
}

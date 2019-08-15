package bridge

import (
    "bifrost/bus"
    "github.com/go-stomp/stomp"
    "github.com/go-stomp/stomp/frame"
    "github.com/google/uuid"
)

type BridgeClientSub struct {
    C           chan *bus.Message // MESSAGE payloads
    E           chan *bus.Message // ERROR payloads.
    Id          *uuid.UUID
    Destination string
    Client      *BridgeClient
    subscribed  bool
}

func (cs *BridgeClientSub) Unsubscribe() {

    unsubscribeFrame := frame.New(frame.UNSUBSCRIBE,
        frame.Id, cs.Id.String(),
        frame.Destination, cs.Destination,
        frame.Ack, stomp.AckAuto.String())

    cs.Client.SendFrame(unsubscribeFrame)
    cs.subscribed = false

}

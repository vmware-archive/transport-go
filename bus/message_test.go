package bus

import (
    "github.com/google/uuid"
    . "github.com/smartystreets/goconvey/convey"
    "testing"
)


func TestMessageModel(t *testing.T) {
    Convey("Test Message model operates correctly", t, func() {
        var message = &Message{
            Id: uuid.New(),
            Payload:  "A new message",
            Channel: "123",
            Direction: Request }
        So(message.Payload, ShouldEqual, "A new message")
        So(message.Direction, ShouldEqual, Request)
        So(message.Channel, ShouldEqual, "123")
    })
}

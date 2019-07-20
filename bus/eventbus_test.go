package bus

import (
    . "github.com/smartystreets/goconvey/convey"
    "testing"
)

func TestEventBus(t *testing.T) {
    Convey("Check we can boot the event bus successfully", t, func() {

        var bus EventBus = new(BifrostEventBus)
        bus.Init();
        Convey("Check the bus boots", func() {
            So(bus.GetChannelManager(), ShouldNotBeNil)
        })
    })
}

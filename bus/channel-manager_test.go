package bus

import (
    . "github.com/smartystreets/goconvey/convey"
    "testing"
)

func TestChannelManager(t *testing.T) {
    Convey("Check channel manager operates correctly", t, func() {
        var manager ChannelManager
        manager = new(ChannelManagerImpl)
        manager.Boot()

        Convey("Check it can be initialized", func() {
            So(len(manager.GetAllChannels()), ShouldEqual, 0)
        })

        Convey("Check we we can create a new channel", func() {
            manager.CreateChannel("melody")
            So(len(manager.GetAllChannels()), ShouldEqual, 1)

            fetchedChannel, _ := manager.GetChannel("melody")
            So(fetchedChannel, ShouldNotBeNil)

            exists := manager.CheckChannelExists("melody")
            So(exists, ShouldBeTrue)
        })

        Convey("Check we get an error when looking for non-existent channels", func() {
            fetchedChannel, err := manager.GetChannel("melody")
            So(fetchedChannel, ShouldBeNil)
            So(err, ShouldNotBeNil)
        })

        Convey("Check we can destroy our new channel ", func() {
            manager.CreateChannel("melody")
            So(len(manager.GetAllChannels()), ShouldEqual, 1)
            manager.DestroyChannel("melody")
            So(len(manager.GetAllChannels()), ShouldEqual, 0)
        })

    })
}

package bus

import (
    . "github.com/smartystreets/goconvey/convey"
    "testing"
)

func TestChannel(t *testing.T) {
    Convey("Check channel operation is as expected", t, func() {

        channel := new(Channel)
        channel.Boot()

        Convey("Check the channel has a stream available", func() {
            So(channel.Stream, ShouldNotBeNil)
        })


        Convey("Check the channel allows sending", func() {
            message := GenerateRequest("test", "hello")
            //
            //go func() {
            //    result := <-channel.Stream
            //    So(result, ShouldNotBeNil)
            //
            //}()


           channel.Send(message)


        })

    })
}

// Copyright 2019 VMware Inc.
package main

import (
    "bifrost/bridge"
    "github.com/go-stomp/stomp"
    "log"
)

var stop = make(chan bool)

var conn *stomp.Conn
var subscribed = make(chan bool)

// these are the default options that work with RabbitMQ
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
    stomp.ConnOpt.Login("guest", "guest"),
    stomp.ConnOpt.Host("/"),
}

func main() {

    // connect to appfabric STOMP over WebSocket
    bc := bridge.NewBrokerConnector()
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: "appfabric.vmware.com:8090",
        WSPath:     "/fabric"}

    c, _ := bc.ConnectWs(config)

    sub := c.WsConn.Subscribe("/topic/simple-stream")

    handler := func() {
        for {
            f := <-sub.C
            log.Println("Message: %s", string(f.Body))
        }
    }

    go handler()

    // connect to local rabbit STOMP over TCP
    rBc := bridge.NewBrokerConnector()
    configR := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: "localhost:61613"}

    rConn, _ := rBc.Connect(configR)

    rConn.Conn.Subscribe("/topic/somewhere", stomp.AckAuto)
    // do something with sSub.C


    //bf := bus.GetBus()
    //channel := "some-channel"
    //bf.GetChannelManager().CreateChannel(channel)
    //
    //// listen for a single request on 'some-channel'
    //requestHandler, _ := bf.ListenRequestStream(channel)
    //requestHandler.Handle(
    //    func(msg *bus.Message) {
    //        pingContent := msg.Payload.(string)
    //        fmt.Printf("\nPing: %s\n", pingContent)
    //
    //        // send a response back.
    //        bf.SendResponseMessage(channel, pingContent, msg.DestinationId)
    //    },
    //    func(err error) {
    //        // something went wrong...
    //    })
    //
    //// send a request to 'some-channel' and handle a single response
    //responseHandler, _ := bf.RequestOnce(channel, "Woo!")
    //responseHandler.Handle(
    //    func(msg *bus.Message) {
    //        fmt.Printf("Pong: %s\n", msg.Payload.(string))
    //    },
    //    func(err error) {
    //        // something went wrong...
    //    })

    // fire the request.

    //go recvMessages(subscribed)

    // conn, _ := stomp.Dial("tcp", *serverAddr, options...)
    // go listen(conn)
    //
    // // wait until we know the receiver has subscribed
    // <-subscribed
    // println("Subscribed!")
    // //go send(conn)
    //

    <-stop

}

// Copyright 2019 VMware Inc.
package main

import (
    "bifrost/bridge"
    "bifrost/bus"
    "encoding/json"
    "flag"
    "fmt"
    "github.com/go-stomp/stomp"
)

const defaultPort = ":61613"

var serverAddr = flag.String("server", "localhost:61613", "STOMP server endpoint")
var queueName = flag.String("queue", "/queue/client_test", "Destination queue")
var msgCount int = 90000
var stop = make(chan bool)

var conn *stomp.Conn
var subscribed = make(chan bool)

// these are the default options that work with RabbitMQ
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
    stomp.ConnOpt.Login("guest", "guest"),
    stomp.ConnOpt.Host("/"),
}

func main() {

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
    // <-stop
    //// <-stop

    bc := bridge.NewBrokerConnector()
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: "localhost:61613"}

    _, err := bc.Connect(config)
    if err != nil {
        fmt.Printf("error connecting %s\n", err)
    }

    sub, err := bc.Subscribe("/topic/wow")

    if err != nil {
        fmt.Printf("error subscribing %s\n", err)
    }
    mC := new(bus.MessageConfig)
    mC.Payload = "Hiya!"

    fmt.Printf("Sending Message %s\n", mC.Payload)

    err = bc.SendMessage("/topic/wow", bus.GenerateRequest(mC))

    if err != nil {
        fmt.Printf("error sending %s\n", err)
    }

    for i := 0; i <= 1; i++ {
        msg := <-sub.StompSub.C
        var pl string
        json.Unmarshal(msg.Body, &pl)
        fmt.Printf("Got Message... %s\n", pl)
    }


}

//func listen(conn *stomp.Conn) {
//
//    defer func() {
//        stop <- true
//    }()
//
//    sub, _ := conn.Subscribe(*queueName, stomp.AckAuto)
//
//    close(subscribed)
//    for i := 1; i <= msgCount; i++ {
//        msg := <-sub.C
//        expectedText := fmt.Sprintf("Happy Chappy Message #%d", i)
//        actualText := string(msg.Body)
//        if expectedText != actualText {
//            println("Expected:", expectedText)
//            println("Actual:", actualText)
//        } else {
//            println("Message we got %s", actualText)
//        }
//    }
//
//
//
//    fmt.Println("listener is now done")
//
//}
//
//func send(conn *stomp.Conn) {
//
//       defer func() {
//           stop <- true
//       }()
//
//       for i := 1; i <= msgCount; i++ {
//           text := fmt.Sprintf("Happy Chappy Message #%d", i)
//           fmt.Printf("sending message: %s", text)
//           conn.Send(*queueName, "text/plain", []byte(text), nil)
//
//       }
//       println("sender finished")
//}

//
//func sendMessages() {
//    defer func() {
//        stop <- true
//    }()
//
//conn, err := stomp.Dial("tcp", *serverAddr, options...)
//    if err != nil {
//        println("cannot connect to server", err.Error())
//        return
//    }
//
//    for i := 1; i <= *messageCount; i++ {
//        text := fmt.Sprintf("Message #%d", i)
//        err = conn.Send(*queueName, "text/plain",
//            []byte(text), nil)
//        if err != nil {
//            println("failed to send to server", err)
//            return
//        }
//    }
//    println("sender finished")
//}
//
//func recvMessages(subscribed chan bool) {
//    defer func() {
//        stop <- true
//    }()
//
//    conn, err := stomp.Dial("tcp", *serverAddr, options...)
//
//    if err != nil {
//        println("cannot connect to server", err.Error())
//        return
//    }
//
//    sub, err := conn.Subscribe(*queueName, stomp.AckAuto)
//    if err != nil {
//        println("cannot subscribe to", *queueName, err.Error())
//        return
//    }
//    close(subscribed)
//
//    for i := 1; i <= *messageCount; i++ {
//        msg := <-sub.C
//        expectedText := fmt.Sprintf("Message #%d", i)
//        actualText := string(msg.Body)
//        if expectedText != actualText {
//            println("Expected:", expectedText)
//            println("Actual:", actualText)
//        }
//    }
//    println("receiver finished")
//
//}

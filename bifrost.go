// Copyright 2019 VMware Inc.
package main

import (
    "bifrost/bridge"
    "flag"
    "fmt"
    "github.com/go-stomp/stomp"
    "github.com/gorilla/websocket"
    "log"
    "net/url"
    "os"
    "os/signal"
    "time"
)

const defaultPort = ":61613"

var serverAddr = flag.String("server", "localhost:15674/ws", "STOMP server endpoint")
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


    /// connect to WS and see how we can combine this stuff.

    var addr = flag.String("addr", "localhost:15674", "http service address")

    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
    log.Printf("connecting to %s", u.String())

    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal("dial:", err)
    }


    defer c.Close()


    //Send("/queue/another-one", "text/plain",
    //    []byte(fmt.Sprintf("Message #%d", i)), nil)

    c.WriteMessage(1, []byte(fmt.Sprintf("CONNECT")))
    c.UnderlyingConn().Write([]byte(fmt.Sprintf("CONNECT")))

    //done := make(chan struct{})
    //
    //go func() {
    //    defer close(done)
    //    for {
    //        _, message, err := c.ReadMessage()
    //        if err != nil {
    //            log.Println("read:", err)
    //            return
    //        }
    //        log.Printf("recv: %s", message)
    //    }
    //}()

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()


    bc := bridge.NewBrokerConnector()
    config := &bridge.BrokerConnectorConfig{
       Username:   "guest",
       Password:   "guest",
       ServerAddr: "localhost:8090/"}




    _, err = bc.ConnectWs(config, c.UnderlyingConn())
    if err != nil {
       fmt.Printf("error connecting %s\n", err)
    }

    <-stop

    //sub, err := bc.Subscribe("/topic/wow")
    //
    //if err != nil {
    //    fmt.Printf("error subscribing %s\n", err)
    //}
    //mC := new(bus.MessageConfig)
    //mC.Payload = "Hiya!"
    //
    //fmt.Printf("Sending Message %s\n", mC.Payload)
    //
    //err = bc.SendMessage("/topic/wow", bus.GenerateRequest(mC))
    //
    //if err != nil {
    //    fmt.Printf("error sending %s\n", err)
    //}
    //
    //for i := 0; i <= 1; i++ {
    //    msg := <-sub.StompSub.C
    //    var pl string
    //    json.Unmarshal(msg.Body, &pl)
    //    fmt.Printf("Got Message... %s\n", pl)
    //}


    //for {
    //    select {
    //    case <-done:
    //        return
    //    case t := <-ticker.C:
    //        err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
    //        if err != nil {
    //            log.Println("write:", err)
    //            return
    //        }
    //    case <-interrupt:
    //        log.Println("interrupt")
    //
    //        // Cleanly close the connection by sending a close message and then
    //        // waiting (with timeout) for the server to close the connection.
    //        err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
    //        if err != nil {
    //            log.Println("write close:", err)
    //            return
    //        }
    //        select {
    //        case <-done:
    //        case <-time.After(time.Second):
    //        }
    //        return
    //    }
    //}


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

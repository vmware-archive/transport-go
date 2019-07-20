package main

import (
    "bifrost/bus"
    "fmt"
    "github.com/google/uuid"
)

var myChan = make(chan bus.Message)
var end = make(chan bool)

func main() {

    go listen()
    sendMessage()
    <- end
}

func sendMessage() {

    msg := &bus.Message{
        Id: uuid.New(),
        Channel: "test",
        Payload: "test msg " }
    myChan <- *msg
}

func listen() {
    message := <- myChan
    payload  := message.Payload.(string)
    fmt.Printf("we got a message! %s", payload)
     end <- true
}

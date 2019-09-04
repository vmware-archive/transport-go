// Copyright 2019 VMware Inc.
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "go-bifrost/bridge"
    "go-bifrost/bus"
    "go-bifrost/model"
    "log"
    "math/rand"
    "os"
    "strconv"
    "time"
)

var addr = ":62134"

func main() {
    var runDemo = flag.Bool("demo", false, "Run Demo - Connect to local service")
    var runService = flag.Bool("service", false, "Run Service - Run local service")
    flag.Parse()

    if *runService {
        fmt.Println("Service Starting...")
        b := bus.GetBus()
        b.StartTCPService(addr)
    }

    if *runDemo {
        runDemoApp()
    }

    if !*runDemo && !*runService {
        fmt.Println("To try things out...")
        flag.PrintDefaults()
        os.Exit(1)
    }

}


func runDemoApp() {

    // get a pointer to the bus.
    b := bus.GetBus()

    // get a pointer to the channel manager
    cm := b.GetChannelManager()

    channel := "my-stream"
    cm.CreateChannel(channel)

    // create done signal
    var done = make(chan bool)

    // listen to stream of messages coming in on channel.
    h, err := b.ListenStream(channel)

    if err != nil {
        log.Panicf("unable to listen to channel stream, error: %e", err)
    }

    count := 0

    // listen for five messages and then exit, send a completed signal on channel.
    h.Handle(
        func(msg *model.Message) {

            // unmarshal the payload into a Response object (used by fabric services)
            r := &model.Response{}
            d := msg.Payload.([]byte)
            json.Unmarshal(d, &r)
            fmt.Printf("Stream ticked from local broker: %s\n", r.Payload.(string))
            count++
            if count >=5 {
                done <- true
            }
        },
        func(err error) {
            log.Panicf("error received on channel %e", err)
        })

    // create a broker connector config, in this case, we will connect to the application fabric demo endpoint.
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: addr }

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        log.Panicf("unable to connect to local broker, error: %e", err)
    }
    fmt.Println("Connected to local broker!")

    // mark our local channel as galactic and map it to our connection and the /topic/simple service
    // running locally
    err = cm.MarkChannelAsGalactic(channel, "/topic/simple", c)
    if err != nil {
        log.Panicf("unable to map local channel to broker destination: %e", err)
    }

    fmt.Printf("\nSending 10 messages to broker, every 500ms\n\n")
    time.Sleep(1 * time.Second)
    for i := 0; i < 10; i++ {

        pl := "ping--" + strconv.Itoa(rand.Intn(10000000))
        r := &model.Response{Payload: pl}
        m, _ := json.Marshal(r)
        c.SendMessage("/topic/simple", m)
        time.Sleep(500 * time.Millisecond)
    }

    // wait for done signal
    <-done

    fmt.Printf("\nDone.\n\n")

    // mark channel as local (unsubscribe from all mappings)
    err = cm.MarkChannelAsLocal(channel)
    if err != nil {
        log.Panicf("unable to unsubscribe, error: %e", err)
    }
    err = c.Disconnect()
    if err != nil {
        log.Panicf("unable to disconnect, error: %e", err)
    }
}

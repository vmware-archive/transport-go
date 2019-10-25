// Copyright 2019 VMware Inc.
package main

import (
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
    "go-bifrost/bridge"
    "go-bifrost/bus"
    "go-bifrost/model"
    "log"
    "math/rand"
    "os"
    "strconv"
    "time"
    "sync"
    "reflect"
    "github.com/urfave/cli"
)

var addr = ":62134"

func main() {
    app := cli.NewApp()
    app.Name = "Bifrost demo app"
    app.Usage = "Demonstrates different features of the Bifrost bus"
    app.Commands = []cli.Command{
        {
            Name: "demo",
            Usage: "Run Demo - Connect to local service",
            Action: func(c *cli.Context) error {
                runDemoApp()
                return nil
            },
        },
        {
            Name: "cal",
            Usage: "Call Calendar service for the time on appfabric.vmware.com",
            Action: func(c *cli.Context) error {
                runDemoCal()
                return nil
            },
        },
        {
            Name: "service",
            Usage: "Run Service - Run local service",
            Action: func(c *cli.Context) error {
                fmt.Println("Service Starting...")
                b := bus.GetBus()
                b.StartTCPService(addr)
                return nil
            },
        },
        {
            Name: "store",
            Usage: "Open galactic store from appfabric.vmware.com",
            Action: func(c *cli.Context) error {
                runDemoStore()
                return nil
            },
        },
    }

    err := app.Run(os.Args)
    if err != nil {
        log.Fatal(err)
    }
}

func runDemoCal() {
    // get a pointer to the bus.
    b := bus.GetBus()

    // get a pointer to the channel manager
    cm := b.GetChannelManager()

    channel := "calendar-service"
    cm.CreateChannel(channel)

    // create done signal
    var done = make(chan bool)

    // listen to stream RESPONSE messages coming in.
    h, err := b.ListenStream(channel)

    if err != nil {
        log.Panicf("unable to listen to channel stream, error: %e", err)
    }

    // handle response from calendar service.
    h.Handle(
        func(msg *model.Message) {

            // unmarshal the payload into a Response object (used by fabric services)
            r := &model.Response{}
            d := msg.Payload.([]byte)
            json.Unmarshal(d, &r)
            fmt.Printf("got time response from service: %s\n", r.Payload.(string))
            done <- true
        },
        func(err error) {
            log.Panicf("error received on channel %e", err)
        })

    // create a broker connector config, in this case, we will connect to the application fabric demo endpoint.
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        UseWS: true,
        WSPath: "/fabric",
        ServerAddr: "appfabric.vmware.com" }

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        log.Panicf("unable to connect to fabric broker, error: %e", err)
    }
    fmt.Println("Connected to fabric broker!")

    // mark our local channel as galactic and map it to our connection and the /topic/calendar-service destination
    err = cm.MarkChannelAsGalactic(channel, "/topic/" + channel, c)
    if err != nil {
        log.Panicf("unable to map local channel to broker destination: %e", err)
    }

    // create request
    id := uuid.New();
    r := &model.Request{}
    r.Request = "time"
    r.Id = &id
    m, _ := json.Marshal(r)
    fmt.Println("Requesting time from calendar service")

    // send request.
    c.SendMessage("/pub/" + channel, m)

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

type SampleMessageItem struct {
    From string `json:"from"`
    Message string `json:"message"`
}

func (mi SampleMessageItem) print() {
    fmt.Println("FROM:", mi.From)
    fmt.Println("Message:", mi.Message)
}

func runDemoStore() {
    // get a pointer to the bus.
    b := bus.GetBus()

    // create a broker connector config, in this case, we will connect to the application fabric demo endpoint.
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        UseWS: true,
        WSPath: "/fabric",
        ServerAddr: "appfabric.vmware.com" }

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        log.Panicf("unable to connect to fabric broker, error: %e", err)
    }
    fmt.Println("Connected to fabric broker:", config.ServerAddr)

    err = b.GetStoreManager().ConfigureStoreSyncChannel(c, "/topic/", "/pub/")
    if err != nil {
        log.Panicf("unable to configure store sync channel, error: %e", err)
    }

    var motdStore bus.BusStore
    motdStore, err = b.GetStoreManager().OpenGalacticStoreWithItemType(
            "messageOfTheDayStore", c, reflect.TypeOf(SampleMessageItem{}))
    if err != nil {
        log.Panicf("failed to open galactic store, error: %e", err)
    }

    wg := sync.WaitGroup{}
    wg.Add(1)
    motdStore.WhenReady(func() {
        wg.Done()
    })

    wg.Wait()

    originalItem := motdStore.GetValue("messageOfTheDay").(SampleMessageItem)
    originalItem.print()

    storeStream := motdStore.OnChange("messageOfTheDay")
    storeStream.Subscribe(func(change *bus.StoreChange) {
        if change.IsDeleteChange {
            fmt.Println("Item removed: ", change.Id)
        } else {
            fmt.Println("Store item changed: ")
            change.Value.(SampleMessageItem).print()
        }
        wg.Done()
    })

    wg.Add(1)
    motdStore.Put("messageOfTheDay",
        SampleMessageItem{
            Message: "updated sample message",
            From: "test user",
        }, "update")
    wg.Wait()

    wg.Add(1)
    motdStore.Put("messageOfTheDay", originalItem, "update")
    wg.Wait()

    // Local stores
    localStringStore := b.GetStoreManager().CreateStore("localStringStore")
    localMessageStore := b.GetStoreManager().CreateStore("localSampleMessageStore")

    // use async transaction to wait for the two local stores.
    tr := b.CreateAsyncTransaction()
    tr.WaitForStoreReady(localStringStore.GetName())
    tr.WaitForStoreReady(localMessageStore.GetName())

    wg.Add(1)
    tr.OnComplete(func(responses []*model.Message) {
        fmt.Println("Local stores initialized")
        fmt.Println("localStringStore:")
        fmt.Println(responses[0].Payload)
        fmt.Println("localSampleMessageStore:")
        fmt.Println(responses[1].Payload)
        wg.Done()
    })

    fmt.Println("Waiting for local stores...")
    tr.Commit()

    localStringStore.Populate(map[string]interface{} {
        "key1": "value1",
        "key2": "value2",
        "key3": "value3",
    })

    // copy the values from the galactic motdStore to the local
    // store
    localMessageStore.Populate(motdStore.AllValuesAsMap())

    b.GetStoreManager().DestroyStore("messageOfTheDayStore")

    wg.Wait()
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

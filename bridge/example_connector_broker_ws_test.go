package bridge_test

import (
    "go-bifrost/bridge"
    "go-bifrost/bus"
    "go-bifrost/model"
    "encoding/json"
    "fmt"
)

func Example_connectUsingBrokerViaWebSocket() {

    // get a reference to the event bus.
    b := bus.GetBus()

    // create a broker connector configuration, using WebSockets.
    config := &bridge.BrokerConnectorConfig{
        Username:   "guest",
        Password:   "guest",
        ServerAddr: "appfabric.vmware.com",
        WSPath:     "/fabric",
        UseWS:      true}

    // connect to broker.
    c, err := b.ConnectBroker(config)
    if err != nil {
        fmt.Printf("unable to connect, error: %e", err)
    }

    // subscribe to our demo simple-stream
    s, _ := c.Subscribe("/topic/simple-stream")

    // set a counter
    n := 0

    // create a control chan
    done := make(chan bool)

    var listener = func() {
        for {
            // listen for incoming messages from subscription.
            m := <-s.GetMsgChannel()

            // unmarshal message.
            r := &model.Response{}
            d := m.Payload.([]byte)
            json.Unmarshal(d, &r)
            fmt.Printf("Message Received: %s\n", r.Payload.(string))

            n++

            // listen for 5 messages then stop.
            if n >= 5 {
                break
            }
        }
        done <- true
    }

    // listen for incoming messages on subscription.
    go listener()

    <-done

    c.Disconnect()
}

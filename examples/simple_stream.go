package examples

import (
	"encoding/json"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"sync"
)

// SimpleStream will connect to our demo broker running at transport-bus.io, listen to a simple stream that is being
// broadcast on /topic/simple-stream. Every second a random word is broadcast on that channel to anyone listening.
// This should take 10 seconds to run.
func SimpleStream() []string {

	// get a pointer to the bus.
	b := bus.GetBus()

	// get a pointer to the channel manager
	cm := b.GetChannelManager()

	channel := "my-stream"
	cm.CreateChannel(channel)

	// listen to stream of messages coming in on channel, a handler is returned that allows you to add in
	// lambdas that handle your
	handler, err := b.ListenStream(channel)

	if err != nil {
		utils.Log.Panicf("unable to listen to channel stream, error: %v", err.Error())
	}

	// create a broker connector config and connect to transport-bus.io over WebSocket using TLS.
	config := &bridge.BrokerConnectorConfig{
		Username:   "guest",            // not required for demo, but our API requires it.
		Password:   "guest",            // ^^ same.
		ServerAddr: "transport-bus.io", // our live broker running plank and demo services.
		UseWS:      true,               // connect over websockets
		WebSocketConfig: &bridge.WebSocketConfig{ // configure websocket
			WSPath: "/ws", // websocket endpoint
			UseTLS: true,  // use TLS/HTTPS
		}}

	// connect to transport-bus.io demo broker
	c, err := b.ConnectBroker(config)
	if err != nil {
		utils.Log.Fatalf("unable to connect to transport-bus.io, error: %v", err.Error())
	}

	// mark our local channel as galactic and map it to our connection and the /topic/simple-stream service
	err = cm.MarkChannelAsGalactic(channel, "/topic/simple-stream", c)
	if err != nil {
		utils.Log.Fatalf("unable to map local channel to broker destination: %e", err)
	}

	// collect the streamed values in a slice
	var streamedValues []string

	// create a wait group that will wait 10 times before completing.
	var wg sync.WaitGroup
	wg.Add(10)

	// keep listening
	handler.Handle(
		func(msg *model.Message) {

			// unmarshal the message payload into a model.Response object
			// this is a wrapper transport uses when being used as a server, it encapsulates a rich set of data
			// about the message, but you only really care about the payload (body)
			r := &model.Response{}
			d := msg.Payload.([]byte)
			err := json.Unmarshal(d, &r)
			if err != nil {
				utils.Log.Errorf("error unmarshalling request, server sent something strange!: %v", err.Error())
				return
			}
			// the value we want is in the payload of our model.Response
			value := r.Payload.(string)

			// log it and save it to our streamedValues
			utils.Log.Infof("stream ticked: %s", value)
			streamedValues = append(streamedValues, value)
			wg.Done()
		},
		func(err error) {
			utils.Log.Errorf("error received on channel %e", err)
		})

	// wait for 10 ticks of the stream, then we're done.
	wg.Wait()

	// close our handler, we're done.
	handler.Close()

	// mark channel as local (unsubscribe from all mappings)
	err = cm.MarkChannelAsLocal(channel)
	if err != nil {
		utils.Log.Fatalf("unable to unsubscribe, error: %e", err)
	}

	// disconnect
	err = c.Disconnect()
	if err != nil {
		utils.Log.Fatalf("unable to disconnect, error: %e", err)
	}

	// return what we got from the stream.
	return streamedValues
}

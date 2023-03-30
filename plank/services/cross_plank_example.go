package services

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/vmware/transport-go/bridge"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"sync"
	"time"
)

const ExternalBrokerExampleServiceChannel = "external-broker-example-service"
const localSyncBusChannel string = "ext-svc"

type ExternalBrokerExampleService struct {
	targetBroker *bridge.BrokerConnectorConfig
	conn         bridge.Connection
	bus          bus.EventBus
	mu           sync.Mutex
}

func NewExternalBrokerExampleService(bus bus.EventBus, config *bridge.BrokerConnectorConfig) *ExternalBrokerExampleService {
	service := &ExternalBrokerExampleService{
		bus:          bus,
		targetBroker: config,
	}
	return service
}

func (s *ExternalBrokerExampleService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	// in this example, I wrote it so that "ping" request assumes the broker session is established already, but in real
	// scenarios you may wish to ensure the connection is alive and healthy. maybe you will want to create a few more
	// service requests to handle the lifecycle of the WS connection to the other broker. for example, you could create
	// a request "connect" for connecting to the external broker and "disconnect" for closing the connection explicitly.
	switch request.Request {
	case "ping":
		// we create a single-fire listener for the channel that's linked to the other broker
		handler, err := s.bus.ListenOnce(localSyncBusChannel)
		if err != nil {
			// failed to set up the listener. this rarely happens though but still it helps to make sure all errors are tracked.
			core.SendErrorResponse(request, 500, err.Error())
			return
		}

		// we will take the payload from the local request and use it to send a remote request in L67
		utils.Log.Infoln("RECEIVED LOCAL REQUEST PAYLOAD", request.Payload)

		// set up the handler for responses returned back from the external broker
		handler.Handle(func(message *model.Message) {
			var payload model.Response
			if err := message.CastPayloadToType(&payload); err != nil {
				utils.Log.Errorln("RECEIVED RESPONSE BUT FAILED TO CAST", err)
				core.SendErrorResponse(request, 500, err.Error())
			}
			// pass back the success response from the external broker back to the requester.
			core.SendResponse(request, payload.Payload)
			utils.Log.Infof("RECEIVED RESPONSE: %v\n", payload.Payload)
		}, func(err error) {
			// external broker sent an error. pass it back to the requester.
			core.SendErrorResponse(request, 400, err.Error())
			utils.Log.Errorln("RECEIVED ERROR RESPONSE FROM EXTERNAL BROKER", err)
			return
		})

		// create a request object for the remote service "ping-pong-service" with request named "ping-get" and pass it
		// through the external broker using the connection we established when the service started. see ping-pong-service
		// L60 for the data structure for the request payload that the service accepts
		req := &model.Request{Request: "ping-get", Payload: request.Payload.(string)}
		reqMarshalled, _ := json.Marshal(req)
		if err = s.conn.SendJSONMessage("/pub/queue/ping-pong-service", reqMarshalled); err != nil {
			core.SendErrorResponse(request, 400, err.Error())
			return
		}

	default:
		core.SendErrorResponse(request, 400, "unknown service")
	}
}

func (s *ExternalBrokerExampleService) OnServiceReady() chan bool {
	ready := make(chan bool, 1)
	s.mu.Lock()
	defer s.mu.Unlock()

	// connect to the external broker
	var err error
	if s.conn, err = s.bus.ConnectBroker(s.targetBroker); err != nil {
		utils.Log.Errorln("[external-broker-example-service] could not connect to the broker. Service failed to start")
		ready <- false
		return ready
	} else {
		utils.Log.Infoln("[external-broker-example-service] connected to external broker!")

		// create a local channel named "ext-svc" and bridge it to the external broker
		busChannelManager := s.bus.GetChannelManager()
		busChannelManager.CreateChannel(localSyncBusChannel)
		if err = s.bus.GetChannelManager().MarkChannelAsGalactic(localSyncBusChannel, "/queue/ping-pong-service", s.conn); err != nil {
			utils.Log.Errorln(err)
			ready <- false
			return ready
		}
	}

	// this is just to demonstrate that a message travels from this Plank process to another.
	// you can use any WebSocket client instead to send a request message to this service and get a response back.
	go func() {
		t := time.NewTicker(5 * time.Second)
		<-t.C
		_ = s.bus.SendRequestMessage(ExternalBrokerExampleServiceChannel, model.Request{Id: &uuid.UUID{}, Request: "ping", Payload: "HI!"}, nil)
	}()

	// at this point, a connection between this Plank and the other broker is established and the local channel "ext-svc"
	// is synced to the bus channel "ping-pong-service" from the other Plank. set the service to ready and return the channel.
	ready <- true
	return ready
}

func (js *ExternalBrokerExampleService) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return nil
}

func (s *ExternalBrokerExampleService) OnServerShutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// disconnect from bridge connection
	if s.conn != nil {
		utils.Log.Infoln("[external-broker-example-service] disconnecting from external broker")
		if err := s.conn.Disconnect(); err != nil {
			utils.Log.Errorln(err)
		}
	}
}

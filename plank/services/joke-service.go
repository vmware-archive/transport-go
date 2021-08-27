package services

import (
	"github.com/google/uuid"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/service"
	"net/http"
	"reflect"
)

const (
	JokeServiceChannel = "joke-service"
)

// Joke is a representation of what is returned by our providing JokeAPI.
type Joke struct {
	Id     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

// JokeService will return a terrible joke, on demand.
type JokeService struct {
	core service.FabricServiceCore
}

// NewJokeService will return an instance of JokeService.
func NewJokeService() *JokeService {
	return &JokeService{}
}

// Init will fire when the service is being registered by the fabric, it passes a reference of the same core
// Passed through when implementing HandleServiceRequest
func (js *JokeService) Init(core service.FabricServiceCore) error {

	// capture a reference to core for later.
	js.core = core

	// JokeService always returns JSON objects as responses.
	core.SetDefaultJSONHeaders()
	return nil
}

// HandleServiceRequest will listen for incoming requests with the command 'get-joke' and will then return a terrible
// Joke back to the requesting component.
func (js *JokeService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	switch request.Request {
	case "get-joke":
		js.getJoke(request, core)
	default:
		core.HandleUnknownRequest(request)
	}
}

// getJoke calls our terrible joke service, and returns the response or error back to the requester.
func (js *JokeService) getJoke(request *model.Request, core service.FabricServiceCore) {

	// make API call using inbuilt RestService to make network calls.
	core.RestServiceRequest(&service.RestServiceRequest{
		Uri:    "https://icanhazdadjoke.com",
		Method: "GET",
		Headers: map[string]string{
			"Accept": "application/json",
		},
		ResponseType: reflect.TypeOf(&Joke{}),
	}, func(response *model.Response) {

		// send back a successful joke.
		core.SendResponse(request, response.Payload.(*Joke))
	}, func(response *model.Response) {

		// something went wrong with the API call, tell the requester.
		fabricError := service.GetFabricError("Get Joke API Call Failed", response.ErrorCode, response.ErrorMessage)
		core.SendErrorResponseWithPayload(request, response.ErrorCode, response.ErrorMessage, fabricError)
	})
}

// GetRESTBridgeConfig returns a config for a REST endpoint for Jokes
func (js *JokeService) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return []*service.RESTBridgeConfig{
		{
			ServiceChannel: JokeServiceChannel,
			Uri:            "/rest/joke",
			Method:         http.MethodGet,
			AllowHead:      true,
			AllowOptions:   true,
			FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
				return model.Request{
					Id:                &uuid.UUID{},
					Request:           "get-joke",
					BrokerDestination: nil,
				}
			},
		},
	}
}

// OnServerShutdown is not implemented in this service.
func (js *JokeService) OnServerShutdown() {}

// OnServiceReady has no functionality in this service, so it returns a fired channel.
func (js *JokeService) OnServiceReady() chan bool {

	// ready right away, nothing to do.
	readyChan := make(chan bool, 1)
	readyChan <- true
	return readyChan
}

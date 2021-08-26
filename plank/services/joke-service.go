package services

import (
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/service"
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
		Uri:          "https://icanhazdadjoke.com",
		Method:       "GET",
		ResponseType: reflect.TypeOf(&Joke{}),
	}, func(response *model.Response) {

		// send back a successful joke.
		core.SendResponse(request, []string{response.Payload.(*Joke).Joke})
	}, func(response *model.Response) {

		// something went wrong with the API call, tell the requester.
		core.SendErrorResponse(request, response.ErrorCode, response.ErrorMessage)
	})
}

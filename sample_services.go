// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package main

import (
    "fmt"
    "github.com/google/uuid"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/service"
    "reflect"
    "time"
)

const (
    CalendarServiceChan = "calendar-service"
    PongServiceChan     = "services-PongService"
    SimpleStreamChan    = "simple-stream"
    ServbotServiceChan  = "servbot"
    VmServiceChan       = "vm-service"
    VMWCloudServiceChan = "services-CloudServiceStatus"
)

type calendarService struct {}

func (cs *calendarService) HandleServiceRequest(
        request *model.Request, core service.FabricServiceCore) {

    switch request.Request {
    case "time":
        core.SendResponse(request, time.Now().Format("03:04:05.000 AM (-0700)"))
    case "date":
        core.SendResponse(request, time.Now().Format("Mon, 02 Jan 2006"))
    default:
        core.HandleUnknownRequest(request)
    }
}

type pongService struct {}

func (cs *pongService) HandleServiceRequest(
        request *model.Request, core service.FabricServiceCore) {

    switch request.Request {
    case "basic":
        core.SendResponse(request, "Fabric Pong (Basic): Pong: " + request.Payload.(string))
    case "full":
        id := uuid.New()
        tr := core.Bus().CreateAsyncTransaction()
        tr.SendRequest(CalendarServiceChan, &model.Request{ Id: &id, Request: "date" })
        tr.SendRequest(CalendarServiceChan, &model.Request{ Id: &id, Request: "time" })
        tr.OnComplete(func(responses []*model.Message) {
            core.SendResponse(request,  fmt.Sprintf("Fabric Pong (Full): Calendar: %s %s / Pong: %s",
                    responses[0].Payload.(*model.Response).Payload,
                    responses[1].Payload.(*model.Response).Payload,
                    request.Payload))
        })
        tr.Commit()
    default:
        core.HandleUnknownRequest(request)
    }
}

type simpleStreamTickerService struct {
    online bool
}

func (fs *simpleStreamTickerService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
    switch request.Request {
    case "online":
        fs.online = true
    case "offline":
        fs.online = false
    default:
        core.HandleUnknownRequest(request)
    }
}

func (fs *simpleStreamTickerService) Init(core service.FabricServiceCore) error {
    fs.online = true
    ticker := time.NewTicker(500 * time.Millisecond)
    go func() {
        for {
            <-ticker.C
            if fs.online {
                now := time.Now()
                id := uuid.New()
                response := &model.Response{
                    Payload: fmt.Sprintf("ping-%d", now.Nanosecond() + now.Second()),
                    Id: &id,
                }
                core.Bus().SendResponseMessage(SimpleStreamChan, response, nil)
            }
        }
    }()
    return nil
}

type Joke struct {
    Id string `json:"id"`
    Joke string `json:"joke"`
    Status int `json:"status"`
}

type servbotService struct {}

func (s *servbotService) Init(core service.FabricServiceCore) error {
    core.SetHeaders(map[string]string{
       "Accept": "application/json",
    })
    return nil
}

func (s *servbotService) HandleServiceRequest(
        request *model.Request, core service.FabricServiceCore) {

    switch request.Request {
    case "Joke":
        core.RestServiceRequest(&service.RestServiceRequest{
            Url:          "https://icanhazdadjoke.com",
            HttpMethod:   "GET",
            ResponseType: reflect.TypeOf(&Joke{}),
        }, func(response *model.Response) {
            core.SendResponse(request, []string {response.Payload.(*Joke).Joke})
        }, func(response *model.Response) {
            core.SendErrorResponse(request, response.ErrorCode, response.ErrorMessage)
        })
    default:
        core.HandleUnknownRequest(request)
    }
}

type vmwCloudServiceService struct {}

func (s *vmwCloudServiceService) HandleServiceRequest(
        request *model.Request, core service.FabricServiceCore) {

    core.RestServiceRequest(&service.RestServiceRequest{
        Url:          "https://status.vmware-services.io/api/v2/status.json",
        HttpMethod:   "GET",
    }, func(response *model.Response) {
        core.SendResponse(request, response.Payload)
    }, func(response *model.Response) {
        core.SendErrorResponse(request, response.ErrorCode, response.ErrorMessage)
    })
}

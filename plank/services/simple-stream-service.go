// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package services

import (
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"math/rand"
	"reflect"
)

const (
	SimpleStreamServiceChannel = "simple-stream"
)

// SimpleStreamService broadcasts a simple random word as a string every 1s on channel simple-stream
// It does not accept any commands and is not listening for them, so there is no point in trying to talk to it.
// It will only broadcast, you can't stop it and it won't be stopped :)
type SimpleStreamService struct {
	words      []string                  // this is a list of random words.
	fabricCore service.FabricServiceCore // reference to the fabric core infrastructure we will need later on.
	readyChan  chan bool                 // once we're ready, let plank know about it.
	cronJob    *cron.Cron                // cronjob that runs every 1s.
}

// NewSimpleStreamService will return a new instance of the SimpleStreamService.
func NewSimpleStreamService() *SimpleStreamService {
	return &SimpleStreamService{}
}

// Init will fire when the service is being registered by the fabric, it passes a reference of the same core
// Passed through when implementing HandleServiceRequest
func (sss *SimpleStreamService) Init(core service.FabricServiceCore) error {

	// capture a reference to our fabric core infrastructure.
	sss.fabricCore = core
	return nil
}

func (sss *SimpleStreamService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	// do nothing in here, we're not listening for any requests.
}

// OnServiceReady will fetch a list of random words, then set up a timer to broadcast every 500ms, forever.
func (sss *SimpleStreamService) OnServiceReady() chan bool {
	sss.readyChan = make(chan bool, 1)
	sss.fetchRandomWords()
	return sss.readyChan
}

// fetchRandomWords will call a public endpoint that very kindly returns random words.
func (sss *SimpleStreamService) fetchRandomWords() {

	restRequest := &service.RestServiceRequest{
		Uri:          "https://random-word-api.herokuapp.com/word?number=500",
		Method:       "GET",
		ResponseType: reflect.TypeOf(sss.words),
	}

	// use the in-build REST Service to make this easy.
	sss.fabricCore.RestServiceRequest(restRequest, sss.handleWordFetchSuccess, sss.handleWordFetchFailure)
}

// handleWordFetchSuccess will parse a successful incoming word response from our API.
func (sss *SimpleStreamService) handleWordFetchSuccess(response *model.Response) {
	sss.words = response.Payload.([]string)
	sss.fireRandomWords()
	sss.readyChan <- true
}

// handleWordFetchFailure will parse a failed random word API request.
func (sss *SimpleStreamService) handleWordFetchFailure(response *model.Response) {
	utils.Log.Info(response.Payload)
	// we have no data, so make something up
	sss.words = []string{"cake", "pizza", "tree", "bunny", "api-failed", "sorry", "rocket", "soda", "burger", "kitty", "fox"}
	sss.fireRandomWords()
	sss.readyChan <- true
}

// fireRandomWords will send out a random word to the stream every second.
func (sss *SimpleStreamService) fireRandomWords() {
	var fireMessage = func() {
		id := uuid.New()
		sss.fabricCore.SendResponse(&model.Request{Id: &id}, sss.getRandomWord())
	}
	sss.cronJob = cron.New()
	sss.cronJob.AddFunc("@every 1s", fireMessage)
	sss.cronJob.Start()
}

// OnServerShutdown will stop the cronjob firing.
func (sss *SimpleStreamService) OnServerShutdown() {
	sss.cronJob.Stop()
}

// getRandomWord will return a random word from the list we retrieved.
func (sss *SimpleStreamService) getRandomWord() string {
	return sss.words[rand.Intn(len(sss.words)-1)]
}

// these lifecycle hooks are not used in this service, bowever we need them to fulfill our contract to enable the
// OnServiceReady lifecycle method
func (sss *SimpleStreamService) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return nil
}

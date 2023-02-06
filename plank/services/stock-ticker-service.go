// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"github.com/vmware/transport-go/stompserver"
	"golang.org/x/net/context/ctxhttp"
)

const (
	StockTickerServiceChannel = "stock-ticker-service"
	StockTickerAPI            = "https://finnhub.io/api/v1/quote"
)

// TickerSnapshotData is the data structure for this demo service
type TickerSnapshotData struct {
	CurrentPrice       float64 `json:"c"`
	Change             float64 `json:"d"`
	PercentChange      float64 `json:"dp"`
	HighDayPrice       float64 `json:"h"`
	LowDayPrice        float64 `json:"l"`
	OpenPrice          float64 `json:"o"`
	PreviousClosePrice float64 `json:"pc"`
	LastUpdated        int64   `json:"t"`
}

// StockTickerService is a more complex real life example where its job is to subscribe clients
// to price updates for a stock symbol. the service accepts a JSON-formatted request from the client
// that must be formatted like this: {"symbol": "<TICKER_SYMBOL>"}.
//
// once the service receives the request, it will schedule a job to query the stock price API
// for the provided symbol, retrieve the data and pipe it back to the client every thirty seconds.
// upon the connected client leaving, the service will remove from its cache the timer.
type StockTickerService struct {
	tickerListenersMap map[string]*time.Ticker
	lock               sync.RWMutex
}

// NewStockTickerService returns a new instance of StockTickerService
func NewStockTickerService() *StockTickerService {
	return &StockTickerService{
		tickerListenersMap: make(map[string]*time.Ticker),
	}
}

// HandleServiceRequest accepts incoming requests and schedules a job to fetch stock price from
// a third party API and return the results back to the user.
func (ps *StockTickerService) HandleServiceRequest(request *model.Request, core service.FabricServiceCore) {
	switch request.Request {
	case "ticker_price_lookup":
		input := request.Payload.(map[string]string)
		response, err := queryStockTickerAPI(input["symbol"])
		if err != nil {
			core.SendErrorResponse(request, 400, err.Error())
			return
		}
		// send the response back to the client
		core.SendResponse(request, response)
		break

	case "ticker_price_update_stream":
		// parse the request and extract user input from key "symbol"
		input := request.Payload.(map[string]interface{})
		symbol := input["symbol"].(string)

		// get the price immediately for the first request
		response, err := queryStockTickerAPI(symbol)
		if err != nil {
			core.SendErrorResponse(request, 400, err.Error())
			return
		}
		// send the response back to the client
		core.SendResponse(request, response)

		// set a ticker that fires every 30 seconds and keep it in a map for later disposal
		ps.subscribeToTickerUpdates(symbol, request, core)
	default:
		core.HandleUnknownRequest(request)
	}
}

// OnServiceReady sets up a listener to monitor the client STOMP sessions disconnecting from
// their sessions so that it can stop the running ticker and destroy it from the map structure
// for individual disconnected clients. this will prevent the service from making unnecessary
// HTTP calls for the clients even after they are gone and also the memory consumed from
// ever growing with each connection.
func (ps *StockTickerService) OnServiceReady() chan bool {
	sessionNotifyHandler, _ := bus.GetBus().ListenStream(bus.STOMP_SESSION_NOTIFY_CHANNEL)
	sessionNotifyHandler.Handle(func(message *model.Message) {
		stompSessionEvt := message.Payload.(*bus.StompSessionEvent)
		if stompSessionEvt.EventType == stompserver.ConnectionClosed ||
			stompSessionEvt.EventType == stompserver.UnsubscribeFromTopic {
			if ticker, exists := ps.tickerListenersMap[stompSessionEvt.Id]; exists {
				ticker.Stop()
				ps.lock.Lock()
				delete(ps.tickerListenersMap, stompSessionEvt.Id)
				ps.lock.Unlock()
				utils.Log.Warnf("timer cleaned for %s. trigger: %v", stompSessionEvt.Id, stompSessionEvt.EventType)
			}
		}
	}, func(err error) {})
	readyChan := make(chan bool, 1)
	readyChan <- true
	return readyChan
}

// OnServerShutdown removes the running tickers
func (ps *StockTickerService) OnServerShutdown() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, ticker := range ps.tickerListenersMap {
		ticker.Stop()
	}
	return
}

// GetRESTBridgeConfig returns a config for a REST endpoint that performs the same action as the STOMP variant
// except that there will be only one response instead of every 30 seconds.
func (ps *StockTickerService) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return []*service.RESTBridgeConfig{
		{
			ServiceChannel: StockTickerServiceChannel,
			Uri:            "/rest/stock-ticker/{symbol}",
			Method:         http.MethodGet,
			AllowHead:      true,
			AllowOptions:   true,
			FabricRequestBuilder: func(w http.ResponseWriter, r *http.Request) model.Request {
				pathParams := mux.Vars(r)
				return model.Request{
					Id:                &uuid.UUID{},
					Payload:           map[string]string{"symbol": pathParams["symbol"]},
					Request:           "ticker_price_lookup",
					BrokerDestination: nil,
				}
			},
		},
	}
}

func (ps *StockTickerService) subscribeToTickerUpdates(symbol string, request *model.Request, core service.FabricServiceCore) {
	ps.lock.Lock()
	ticker, hasSubscription := ps.tickerListenersMap[request.BrokerDestination.ConnectionId]

	// if the client already subbed for ticker updates, cancel the timer first to prevent memory leaks
	if hasSubscription {
		utils.Log.Infoln("clearing an existing sub for client")
		ticker.Stop()
	}
	ticker = time.NewTicker(30 * time.Second)
	ps.tickerListenersMap[request.BrokerDestination.ConnectionId] = ticker
	ps.lock.Unlock()

	var response map[string]any
	var err error

	// set up a handler for every time a ticker fires.
	go func() {
		for {
			select {
			case <-ticker.C:
				response, err = queryStockTickerAPI(symbol)
				if err != nil {
					core.SendErrorResponse(request, 500, err.Error())
					continue
				}

				// log message to demonstrate that once the client disconnects
				// the server disposes of the ticker to prevent memory leak.
				utils.Log.Infoln("sending...")

				// send the response back to the client
				core.SendResponse(request, response)
			}
		}
	}()
}

// newTickerRequest is a convenient function that takes symbol as an input and returns
// a new HTTP request object along with any error
func newTickerRequest(symbol string) (*http.Request, error) {
	uv := url.Values{}
	uv.Set("symbol", symbol)

	req, err := http.NewRequest("GET", StockTickerAPI, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = uv.Encode()
	req.Header.Set("X-Finnhub-Token", "sandbox_c4l951aad3iftk6rfja0")
	return req, nil
}

// queryStockTickerAPI performs an HTTP request against the Stock Ticker API and returns the results
// as a generic map[string]interface{} structure. if there's any error during the request-response cycle
// a nil will be returned followed by an error object.
func queryStockTickerAPI(symbol string) (map[string]interface{}, error) {
	// craft a new HTTP request for the stock price provider API
	req, err := newTickerRequest(symbol)
	if err != nil {
		return nil, err
	}

	// perform an HTTP call
	rsp, err := ctxhttp.Do(context.Background(), http.DefaultClient, req)
	if err != nil {
		return nil, err
	}

	// parse the response from the HTTP call
	defer rsp.Body.Close()
	tickerData := &TickerSnapshotData{}
	b, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, tickerData); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"symbol":             symbol,
		"lastRefreshed":      time.Unix(tickerData.LastUpdated, 0).String(),
		"currentPrice":       tickerData.CurrentPrice,
		"change":             tickerData.Change,
		"highDayPrice":       tickerData.HighDayPrice,
		"lowDayPrice":        tickerData.LowDayPrice,
		"openPrice":          tickerData.OpenPrice,
		"closePrice":         tickerData.PreviousClosePrice, // for backward compatible with UI examples
		"previousClosePrice": tickerData.PreviousClosePrice,
	}, nil
}

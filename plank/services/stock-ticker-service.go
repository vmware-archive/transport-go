// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package services

import (
	"context"
	"encoding/json"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"github.com/vmware/transport-go/stompserver"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	StockTickerServiceChannel = "stock-ticker-service"
	StockTickerAPI = "https://www.alphavantage.co/query"
)

// TickerSnapshotData and TickerMetadata ares the data structures for this demo service
type TickerSnapshotData struct {
	MetaData *TickerMetadata `json:"Meta Data"`
	TimeSeries map[string]map[string]interface{} `json:"Time Series (1min)"`
}

type TickerMetadata struct {
	Information string `json:"1. Information"`
	Symbol string `json:"2. Symbol"`
	LastRefreshed string `json:"3. Last Refreshed"`
	Interval string `json:"4. Interval"`
	OutputSize string `json:"5. Output Size"`
	TimeZone string `json:"6. Time Zone"`
}

// StockTickerService is a more complex real life example where its job is to subscribe clients
// to price updates for a stock symbol. the service accepts a JSON-formatted request from the client
// that must be formatted like this: {"symbol": "<TICKER_SYMBOL>"}.
//
// once the service receives the request, it will schedule a job to query the stock price API
// for the provided symbol, retrieve the data and pipe it back to the client every thirty seconds.
// upon the connected client leaving, the service will remove from its cache the timer.
type StockTickerService struct{
	tickerListenersMap map[string]*time.Ticker
	lock sync.RWMutex
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
	case "receive_ticker_updates":
		// parse the request and extract user input from key "symbol"
		input := request.Payload.(map[string]interface{})
		symbol := input["symbol"].(string)

		// set a ticker that fires every 30 seconds and keep it in a map for later disposal
		ps.lock.Lock()
		ticker := time.NewTicker(30 * time.Second)
		ps.tickerListenersMap[request.BrokerDestination.ConnectionId] = ticker
		ps.lock.Unlock()

		// set up a handler for every time a ticker fires.
		go func() {
			for {
				select {
				case <-ticker.C:
					// craft a new HTTP request for the stock price provider API
					req, err := newTickerRequest(symbol)
					if err != nil {
						core.SendErrorResponse(request, 400, err.Error())
						continue
					}

					// perform an HTTP call
					rsp, err := ctxhttp.Do(context.Background(), http.DefaultClient, req)
					if err != nil {
						core.SendErrorResponse(request, rsp.StatusCode, err.Error())
						continue
					}

					// parse the response from the HTTP call
					defer rsp.Body.Close()
					tickerData := &TickerSnapshotData{}
					b, err := ioutil.ReadAll(rsp.Body)
					if err != nil {
						core.SendErrorResponse(request, 500, err.Error())
						continue
					}

					if err = json.Unmarshal(b, tickerData); err != nil {
						core.SendErrorResponse(request, 500, err.Error())
						continue
					}

					if tickerData == nil || tickerData.TimeSeries == nil {
						core.SendErrorResponse(request, 500, string(b))
						continue
					}

					// extract the data we need.
					latestClosePriceStr := tickerData.TimeSeries[tickerData.MetaData.LastRefreshed]["4. close"].(string)
					latestClosePrice, err := strconv.ParseFloat(latestClosePriceStr, 32)
					if err != nil {
						core.SendErrorResponse(request, 500, err.Error())
						continue
					}

					// log message to demonstrate that once the client disconnects
					// the server disposes of the ticker to prevent memory leak.
					utils.Log.Warnln("sending...")

					// send the response back to the client
					core.SendResponse(request, map[string]interface{}{
						"symbol": symbol,
						"lastRefreshed": tickerData.MetaData.LastRefreshed,
						"closePrice": latestClosePrice,
					})
				}
			}
		}()
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
		if stompSessionEvt.EventType == stompserver.ConnectionClosed {
			if ticker, exists := ps.tickerListenersMap[stompSessionEvt.Id]; exists {
				ticker.Stop()
				ps.lock.Lock()
				delete(ps.tickerListenersMap, stompSessionEvt.Id)
				ps.lock.Unlock()
				utils.Log.Warnln("client", stompSessionEvt.Id, "disconnected")
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

// GetRESTBridgeConfig returns nothing. this service is only available through
// STOMP over WebSocket.
func (ps *StockTickerService) GetRESTBridgeConfig() []*service.RESTBridgeConfig {
	return nil
}

// newTickerRequest is a convenient function that takes symbol as an input and returns
// a new HTTP request object along with any error
func newTickerRequest(symbol string) (*http.Request, error) {
	uv := url.Values{}
	uv.Set("function", "TIME_SERIES_INTRADAY")
	uv.Set("symbol", symbol)
	uv.Set("interval", "1min")
	uv.Set("apikey", "XPVMLSLINKN27RWA")

	req, err := http.NewRequest("GET", StockTickerAPI, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = uv.Encode()
	return req, nil
}
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/utils"
	"github.com/vmware/transport-go/service"
	"net/http"
	"strings"
	"time"
)

// buildEndpointHandler builds a http.HandlerFunc that wraps Transport Bus operations in an HTTP request-response cycle.
// service channel, request builder and rest bridge timeout are passed as parameters.
func buildEndpointHandler(svcChannel string, reqBuilder service.RequestBuilder, restBridgeTimeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Errorln(r)
				http.Error(w, "Internal Server Error", 500)
			}
		}()

		// set context that expires after the provided amount of time in restBridgeTimeout to prevent requests from hanging forever
		ctx, cancelFn := context.WithTimeout(context.Background(), restBridgeTimeout)
		defer cancelFn()
		h, err := bus.GetBus().ListenOnce(svcChannel)
		if err != nil {
			panic(err)
		}

		// set up a channel through which to receive the raw response from transport channel
		// handler function runs in another thread so we need to utilize channel to use the correct writer.
		chanReturn := make(chan *transportChannelResponse)
		h.Handle(func(message *model.Message) {
			chanReturn <- &transportChannelResponse{message: message}
		}, func(err error) {
			chanReturn <- &transportChannelResponse{err: err}
		})

		// relay the request to transport channel
		reqModel := reqBuilder(w, r)
		err = bus.GetBus().SendRequestMessage(svcChannel, reqModel, reqModel.Id)

		// get a response from the channel, render the results using ResponseWriter and log the data/error
		// to the console as well.
		select {
		case <-ctx.Done():
			http.Error(
				w,
				fmt.Sprintf("No response received from service channel in %s, request timed out", restBridgeTimeout.String(),
			), 500)
		case chanResponse := <-chanReturn:
			if chanResponse.err != nil {
				utils.Log.WithError(chanResponse.err).Errorf(
					"Error received from channel %s:", svcChannel)
				http.Error(w, chanResponse.err.Error(), 500)
			} else {
				// only send the actual user payload not wrapper information
				response := chanResponse.message.Payload.(*model.Response)
				var respBody interface{}
				if response.Error {
					if response.Payload != nil {
						respBody = response.Payload
					} else {
						respBody = response
					}
				} else {
					respBody = response.Payload
				}

				utils.Log.WithFields(logrus.Fields{
					//"payload": respBody, // don't show this, we may be sending around big byte arrays
				}).Debugf("Response received from channel %s:", svcChannel)

				// if our message is an error and it has a code, lets send that back to the client.
				if response.Error {

					// we have to set the headers for the error response
					for k, v := range response.Headers {
						w.Header().Set(k, v)
					}

					// deal with the response body now, if set.
					n, e := json.Marshal(respBody)
					if e != nil {

						w.WriteHeader(response.ErrorCode)
						w.Write([]byte(response.ErrorMessage))
						return

					} else {
						w.WriteHeader(response.ErrorCode)
						w.Write(n)
						return
					}
				} else {
					// if the response has headers, set those headers. particularly if you're sending around
					// byte array data for things like zip files etc.
					for k, v := range response.Headers {
						w.Header().Set(k, v)
						if strings.ToLower(k) == "content-type" {
							respBody, err = utils.ConvertInterfaceToByteArray(v, respBody)
						}
					}

					var respBodyBytes []byte
					// ensure respBody is properly converted to a byte array as Content-Type header might not be
					// set in the request and the restBody could be in a format that can be json marshalled.
					respBodyBytes, err = marshalResponseBody(respBody)

					// write the non-error payload back.
					if _, err = w.Write(respBodyBytes); err != nil {
						utils.Log.WithError(err).Errorf("Error received from channel %s:", svcChannel)
						http.Error(w, err.Error(), 500)
					}
				}
			}
		}
	}
}


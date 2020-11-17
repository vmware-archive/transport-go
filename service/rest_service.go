// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
	"bytes"
	"encoding/json"
	"gitlab.eng.vmware.com/bifrost/go-bifrost/model"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

const (
	restServiceChannel = "fabric-rest"
)

type RestServiceRequest struct {
	// The destination URL of the request.
	Uri string `json:"uri"`
	// HTTP Method to use, e.g. GET, POST, PATCH etc.
	Method string `json:"method"`
	// The body of the request. String and []byte payloads will be sent as is,
	// all other payloads will be serialized as json.
	Body interface{} `json:"body"`
	//  HTTP headers of the request.
	Headers map[string]string `json:"headers"`
	// Optional type of the response body. If provided the service will try to deserialize
	// the response to this type.
	// If omitted the response body will be deserialized as map[string]interface{}
	// Note that if the response body is not a valid json you should set
	// the ResponseType to string or []byte otherwise you might get deserialization error
	// or empty result.
	ResponseType reflect.Type
	// Shouldn't be populated directly, the field is used to deserialize
	// com.vmware.bifrost.core.model.RestServiceRequest Java/Typescript requests
	ApiClass string `json:"apiClass"`
}

func (request *RestServiceRequest) marshalBody() ([]byte, error) {
	// don't marshal string and []byte payloads as json
	stringPayload, ok := request.Body.(string)
	if ok {
		return []byte(stringPayload), nil
	}
	bytePayload, ok := request.Body.([]byte)
	if ok {
		return bytePayload, nil
	}
	// encode the message payload as JSON
	return json.Marshal(request.Body)
}

type restService struct {
	httpClient http.Client
	baseHost   string
}

func (rs *restService) setBaseHost(host string) {
	rs.baseHost = host
}

func (rs *restService) HandleServiceRequest(request *model.Request, core FabricServiceCore) {

	restReq, ok := rs.getRestServiceRequest(request)
	if !ok {
		core.SendErrorResponse(request, 500, "invalid RestServiceRequest payload")
		return
	}

	body, err := restReq.marshalBody()
	if err != nil {
		core.SendErrorResponse(request, 500, "cannot marshal request body: "+err.Error())
		return
	}

	httpReq, err := http.NewRequest(restReq.Method,
		rs.getRequestUrl(restReq.Uri, core), bytes.NewBuffer(body))

	if err != nil {
		core.SendErrorResponse(request, 500, err.Error())
		return
	}

	// update headers
	for k, v := range restReq.Headers {
		httpReq.Header.Add(k, v)
	}

	// add default Content-Type header if such is not provided in the request
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Add("Content-Type", "application/merge-patch+json")
	}

	httpResp, err := rs.httpClient.Do(httpReq)
	if err != nil {
		core.SendErrorResponse(request, 500, err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 300 {
		core.SendErrorResponseWithPayload(request, httpResp.StatusCode,
			"rest-service error, unable to complete request: "+httpResp.Status,
			map[string]interface{}{
				"errorCode": httpResp.StatusCode,
				"message":   "rest-service error, unable to complete request: " + httpResp.Status,
			})
		return
	}

	result, err := rs.deserializeResponse(httpResp.Body, restReq.ResponseType)
	if err != nil {
		core.SendErrorResponse(request, 500, "failed to deserialize response:"+err.Error())
	} else {
		core.SendResponse(request, result)
	}
}

func (rs *restService) getRestServiceRequest(request *model.Request) (*RestServiceRequest, bool) {
	restReq, ok := request.Payload.(*RestServiceRequest)
	if ok {
		return restReq, true
	}

	// check if the request.Payload is map[string]interface{} and convert it to RestServiceRequest
	// This is needed to handle requests coming from Java/Typescript Transport clients.
	reqAsMap, ok := request.Payload.(map[string]interface{})
	if ok {
		restServReqInt, err := model.ConvertValueToType(reqAsMap, reflect.TypeOf(&RestServiceRequest{}))
		if err == nil && restServReqInt != nil {
			restServReq := restServReqInt.(*RestServiceRequest)
			if restServReq.ApiClass == "java.lang.String" {
				restServReq.ResponseType = reflect.TypeOf("")
			}
			return restServReq, true
		}
	}

	return nil, false
}

func (rs *restService) getRequestUrl(address string, core FabricServiceCore) string {
	if rs.baseHost == "" {
		return address
	}

	result, err := url.Parse(address)
	if err != nil {
		return address
	}
	result.Host = rs.baseHost
	return result.String()
}

func (rs *restService) deserializeResponse(
	body io.ReadCloser, responseType reflect.Type) (interface{}, error) {

	if responseType != nil {

		// check for string responseType
		if responseType.Kind() == reflect.String {
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(body)
			if err != nil {
				return nil, err
			}
			return buf.String(), nil
		}

		// check for []byte responseType
		if responseType.Kind() == reflect.Slice &&
			responseType == reflect.TypeOf([]byte{}) {
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(body)
			if err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}

		var returnResultAsPointer bool
		if responseType.Kind() == reflect.Ptr {
			returnResultAsPointer = true
			responseType = responseType.Elem()
		}
		decodedValuePtr := reflect.New(responseType).Interface()
		err := json.NewDecoder(body).Decode(&decodedValuePtr)
		if err != nil {
			return nil, err
		}
		if returnResultAsPointer {
			return decodedValuePtr, nil
		} else {
			return reflect.ValueOf(decodedValuePtr).Elem().Interface(), nil
		}
	} else {
		var result map[string]interface{}
		err := json.NewDecoder(body).Decode(&result)
		return result, err
	}
}

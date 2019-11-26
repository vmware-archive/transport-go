// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
    "bytes"
    "encoding/json"
    "errors"
    "github.com/stretchr/testify/assert"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
    "io/ioutil"
    "net/http"
    "reflect"
    "strings"
    "sync"
    "testing"
)

type testItem struct {
    Name string `json:"name"`
    Count int `json:"count"`
}

func TestRestServiceRequest_marshalBody(t *testing.T) {
    reqWithStringBody := &RestServiceRequest{Body: "test-body"}
    body, err := reqWithStringBody.marshalBody()
    assert.Nil(t, err)
    assert.Equal(t, []byte("test-body"), body)

    reqWithBytesBody := &RestServiceRequest{Body: []byte{1,2,3,4}}
    body, err = reqWithBytesBody.marshalBody()
    assert.Nil(t, err)
    assert.Equal(t, reqWithBytesBody.Body, body)

    item := testItem{Name: "test-name", Count: 5}
    reqWithTestItem := &RestServiceRequest{Body: item}
    body, err = reqWithTestItem.marshalBody()
    assert.Nil(t, err)
    expectedValue, _ := json.Marshal(item)
    assert.Equal(t, expectedValue, body)
}

func TestRestService_AutoRegistration(t *testing.T) {
    assert.NotNil(t, GetServiceRegistry().(*serviceRegistry).services[restServiceChannel])
}

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) (*http.Response, error)

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
    return f(req)
}


func TestRestService_HandleServiceRequest(t *testing.T) {
    core := newTestFabricCore(restServiceChannel)

    restService := &restService{}
    var lastHttpRequest *http.Request
    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        lastHttpRequest = req
        return &http.Response{
            StatusCode: 200,
            Body: ioutil.NopCloser(bytes.NewBufferString("test-response-body")),
            Header: make(http.Header),
        }, nil
    })

    var lastResponse model.Response

    wg := sync.WaitGroup{}
    wg.Add(1)

    mh, _  := core.Bus().ListenStream(restServiceChannel)
    mh.Handle(
        func(message *model.Message) {
            lastResponse = message.Payload.(model.Response)
            wg.Done()
        },
        func(e error) {
            assert.Fail(t, "unexpected error")
        })

    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            Headers: map[string]string{ "header1": "value1", "header2": "value2"},
            HttpMethod: "UPDATE",
            Body: "test-body",
            ResponseType: reflect.TypeOf(""),
        },
    }, core)

    wg.Wait()

    assert.NotNil(t, lastHttpRequest)
    assert.Equal(t, lastHttpRequest.URL.String(), "http://localhost:4444/test-url")
    assert.Equal(t, lastHttpRequest.Method, "UPDATE")
    assert.Equal(t, lastHttpRequest.Header.Get("header1"), "value1")
    assert.Equal(t, lastHttpRequest.Header.Get("header2"), "value2")
    assert.Equal(t, lastHttpRequest.Header.Get("Content-Type"), "application/merge-patch+json")
    sentBody, _ := ioutil.ReadAll(lastHttpRequest.Body)
    assert.Equal(t, sentBody, []byte("test-body"))

    assert.NotNil(t, lastResponse)
    assert.Equal(t, lastResponse.Payload, "test-response-body")
    assert.False(t, lastResponse.Error)

    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        lastHttpRequest = req
        return &http.Response{
            StatusCode: 200,
            Body: ioutil.NopCloser(bytes.NewBufferString(`{"name": "test-name", "count": 2}`)),
            Header: make(http.Header),
        }, nil
    })

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            Headers: map[string]string {"Content-Type": "json"},
            ResponseType: reflect.TypeOf(testItem{}),
        },
    }, core)

    wg.Wait()

    assert.Equal(t, lastHttpRequest.Header.Get("Content-Type"), "json")
    assert.Equal(t, lastResponse.Payload, testItem{Name:"test-name", Count: 2})

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            ResponseType: reflect.TypeOf(&testItem{}),
        },
    }, core)

    wg.Wait()

    assert.Equal(t, lastResponse.Payload, &testItem{Name:"test-name", Count: 2})

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
        },
    }, core)

    wg.Wait()

    assert.Equal(t, lastResponse.Payload, map[string]interface{} {"name": "test-name", "count": float64(2)})


    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        lastHttpRequest = req
        return &http.Response{
            StatusCode: 200,
            Body: ioutil.NopCloser(bytes.NewBuffer([]byte{1,2,3,4,5})),
            Header: make(http.Header),
        }, nil
    })

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            ResponseType: reflect.TypeOf([]byte{}),
        },
    }, core)

    wg.Wait()

    assert.Equal(t, lastResponse.Payload, []byte{1,2,3,4,5})
}

func TestRestService_HandleServiceRequest_InvalidInput(t *testing.T) {
    core := newTestFabricCore(restServiceChannel)

    restService := &restService{}
    var lastResponse model.Response

    wg := sync.WaitGroup{}
    wg.Add(1)
    mh, _  := core.Bus().ListenStream(restServiceChannel)
    mh.Handle(
        func(message *model.Message) {
            lastResponse = message.Payload.(model.Response)
            wg.Done()
        },
        func(e error) {
            assert.Fail(t, "unexpected error")
        })

    restService.HandleServiceRequest(&model.Request{
        Payload: RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            HttpMethod: "UPDATE",
        },
    }, core)

    wg.Wait()

    assert.NotNil(t, lastResponse)
    assert.True(t, lastResponse.Error)
    assert.Equal(t, lastResponse.ErrorCode, 500)
    assert.Equal(t, lastResponse.ErrorMessage, "invalid RestServiceRequest payload")

    wg.Add(1)

    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            HttpMethod: "@!#$%^&**()",
        },
    }, core)

    wg.Wait()
    assert.True(t, lastResponse.Error)
    assert.Equal(t, lastResponse.ErrorCode, 500)

    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        return nil, errors.New("custom-rest-error")
    })

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
        },
    }, core)
    wg.Wait()

    assert.True(t, lastResponse.Error)
    assert.Equal(t, lastResponse.ErrorCode, 500)
    assert.True(t, strings.Contains(lastResponse.ErrorMessage, "custom-rest-error"))

    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        return &http.Response{
            StatusCode: 404,
            Body: ioutil.NopCloser(bytes.NewBufferString("error-response")),
            Header: make(http.Header),
        }, nil
    })

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
        },
    }, core)
    wg.Wait()

    assert.True(t, lastResponse.Error)
    assert.Equal(t, lastResponse.ErrorCode, 404)
    assert.Equal(t, lastResponse.ErrorMessage, "rest-service error, unable to complete request: error-response")

    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        return &http.Response{
            StatusCode: 200,
            Body: ioutil.NopCloser(bytes.NewBufferString("}")),
            Header: make(http.Header),
        }, nil
    })

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
            ResponseType: reflect.TypeOf(&testItem{}),
        },
    }, core)
    wg.Wait()

    assert.True(t, lastResponse.Error)
    assert.Equal(t, lastResponse.ErrorCode, 500)
    assert.True(t, strings.HasPrefix(lastResponse.ErrorMessage, "failed to deserialize response:"))
}

func TestRestService_setBaseHost(t *testing.T) {
    core := newTestFabricCore(restServiceChannel)
    restService := &restService{}

    wg := sync.WaitGroup{}


    var lastHttpRequest *http.Request
    restService.httpClient.Transport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
        lastHttpRequest = req
        wg.Done()
        return &http.Response{
            StatusCode: 200,
            Body: ioutil.NopCloser(bytes.NewBufferString("test-response-body")),
            Header: make(http.Header),
        }, nil
    })

    restService.setBaseHost("appfabric.vmware.com:9999")

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
        },
    }, core)

    wg.Wait()

    assert.Equal(t, lastHttpRequest.Host, "appfabric.vmware.com:9999")


    restService.setBaseHost("")

    wg.Add(1)
    restService.HandleServiceRequest(&model.Request{
        Payload: &RestServiceRequest{
            Url: "http://localhost:4444/test-url",
        },
    }, core)
    wg.Wait()

    assert.Equal(t, lastHttpRequest.Host, "localhost:4444")
}



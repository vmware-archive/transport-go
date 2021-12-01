package server

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBaseError_Is_errServerInit(t *testing.T) {
	e := wrapError(errServerInit, errors.New("some init fail"))
	assert.True(t, errors.Is(e, errServerInit))
}

func TestBaseError_Error_errServerInit(t *testing.T) {
	e := wrapError(errServerInit, errors.New("some init fail"))
	assert.EqualValues(t, "[plank] Error: Server initialization failed: some init fail\n", e.Error())
}

func TestBaseError_Is_errInternal(t *testing.T) {
	e := wrapError(errInternal, errors.New("internal server error"))
	assert.True(t, errors.Is(e, errInternal))
}

func TestBaseError_Error_errInternal(t *testing.T) {
	e := wrapError(errInternal, errors.New("internal server error"))
	assert.EqualValues(t, "[plank] Error: Internal error: internal server error\n", e.Error())
}

func TestBaseError_Is_errHttp(t *testing.T) {
	e := wrapError(errHttp, errors.New("404"))
	assert.True(t, errors.Is(e, errHttp))
}

func TestBaseError_Error_errHttp(t *testing.T) {
	e := wrapError(errHttp, errors.New("404"))
	assert.EqualValues(t, "[plank] Error: HTTP error: 404\n", e.Error())
}

func TestBaseError_Is_undefined(t *testing.T) {
	e := wrapError(errors.New("some random stuff"), errors.New("?"))
	assert.True(t, errors.Is(e, errUndefined))
}

func TestBaseError_Error_undefined(t *testing.T) {
	e := wrapError(errors.New("some random stuff"), errors.New("?"))
	assert.EqualValues(t, "[plank] Error: Undefined error: ?\n", e.Error())
}

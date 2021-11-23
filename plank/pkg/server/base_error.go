package server

import "fmt"

var (
	errServerInit = &baseError{message: "Server initialization failed"}
	errHttp       = &baseError{message: "HTTP error"}
	errInternal   = &baseError{message: "Internal error"}
	errUndefined  = &baseError{message: "Undefined error"}
)

type baseError struct {
	wrappedErr error
	baseErr    *baseError
	message    string
}

func (e baseError) Is(err error) bool {
	return e.baseErr == err
}

func (e baseError) Error() string {
	return fmt.Sprintf("[plank] Error: %s: %s\n", e.baseErr.message, e.wrappedErr.Error())
}

func wrapError(baseType error, err error) error {
	switch baseType {
	case errServerInit:
		return &baseError{baseErr: errServerInit, wrappedErr: err}
	case errInternal:
		return &baseError{baseErr: errInternal, wrappedErr: err}
	case errHttp:
		return &baseError{baseErr: errHttp, wrappedErr: err}
	}
	return &baseError{baseErr: errUndefined, wrappedErr: err}
}

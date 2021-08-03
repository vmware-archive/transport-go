package utils

import (
	"encoding/json"
	"strings"
)

// ConvertInterfaceToByteArray converts the interface i into a byte array. Depending on the
// value of mimeType being of JSON type, either JSON Marshaller is used or the interface is
// just straight cast to a byte array.
func ConvertInterfaceToByteArray(mimeType string, i interface{}) (results []byte, err error) {
	// use JSON Marshaller for application/json mime type
	if strings.Contains(mimeType, "json") {
		results, err = json.Marshal(i)
		return
	}

	// for the rest mime types, cast the original data format to a byte array
	switch i.(type) {
	case string:
		results = []byte(i.(string))
		break
	default:
		results = i.([]byte)
	}
	return
}

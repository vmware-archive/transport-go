package service

// FabricError is a RFC7807 standard error properties (https://tools.ietf.org/html/rfc7807)
type FabricError struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// GetFabricError will return a structured, standardized Error object that is compliant
// with RFC7807 standard error properties (https://tools.ietf.org/html/rfc7807)
func GetFabricError(message string, code int, detail string) FabricError {
	return FabricError{
		Title:  message,
		Status: code,
		Detail: detail,
		Type:   "https://github.com/vmware/transport-go/blob/main/plank/services/fabric_error.md",
	}
}

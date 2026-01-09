package types

import "time"

type StandardResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	TraceID   string      `json:"traceId"`
}

func NewSuccessResponseWithCode(data interface{}, code int, message string) *StandardResponse {
	return &StandardResponse{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}
func NewErrorResponse(code int, message string, errorDetails ...string) *StandardResponse {
	resp := &StandardResponse{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
	if len(errorDetails) > 0 {
		resp.Data = map[string]interface{}{"error_details": errorDetails[0]}
	}
	return resp
}

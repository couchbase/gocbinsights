package httpqueryclient

import "encoding/json"

type jsonInsightsError struct {
	Code  uint32 `json:"code"`
	Msg   string `json:"msg"`
	Retry bool   `json:"retriable"`
}

type jsonInsightsErrorResponse struct {
	Errors json.RawMessage `json:"errors"`
}

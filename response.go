package idem

import "net/http"

// Response represents a cached HTTP response.
type Response struct {
	StatusCode int         `json:"status_code"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"body"`
}

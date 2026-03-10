package idem

// Response represents a cached HTTP response.
type Response struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}

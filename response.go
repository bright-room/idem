package idem

// Response はキャッシュされるHTTPレスポンスを表す。
type Response struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}

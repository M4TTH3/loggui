package server

import "net/http"

type context struct {
	*http.Request
	http.ResponseWriter

	responseHeader http.Header
	requestHeader func() http.Header
}

func newContext(w http.ResponseWriter, r *http.Request) *context {
	return &context{
		Request:        r,
		ResponseWriter: w,
		responseHeader: r.Header,
		requestHeader:  w.Header,
	}
}

type ctxHandler interface {
	serveHTTP(*context)
}

type ctxHandlerFunc func(*context)

func (f ctxHandlerFunc) serveHTTP(c *context) {
	f(c)
}

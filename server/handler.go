package server

import "net/http"

// mux is a wrapper implementation of ServeMux which allows for middleware
// to be applied to each request.
//
// Implements http.Handler
type mux struct {
	*http.ServeMux
	middlewares []middleware
}

// newHandler returns an http handler for the server
func newMux() *mux {
	return &mux{
		ServeMux: http.NewServeMux(),
	}
}

func (h *mux) use(m middleware) {
	h.middlewares = append(h.middlewares, m)
}

func (h *mux) handle(pattern string, handler ctxHandler) {
	wrapHandler := handler

	for _, m := range h.middlewares {
		wrapHandler = m.wrap(wrapHandler)
	}

	// Use the default ServeMux to handle the request
	// and pass the context to the handler
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := newContext(w, r)
		wrapHandler.serveHTTP(c)
	})

	h.ServeMux.Handle(pattern, httpHandler)
}

func (h *mux) handleFunc(pattern string, handlerFunc ctxHandlerFunc) {
	h.handle(pattern, handlerFunc)
}

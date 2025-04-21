package server

import "net/http"

// middleware is an interface to wrap http handlers with middleware.
type middleware interface {

	// wrap wraps the next http handler with the middleware.
	// It should call next.serveHTTP() to continue the chain.
	wrap(next ctxHandler) ctxHandler
}

// BasicAuthMiddleware is a middleware that adds basic authentication to the handler.
type basicAuthMiddleware struct {
	username string
	password string
}

func newBasicAuthMiddleware(username, password string) *basicAuthMiddleware {
	return &basicAuthMiddleware{
		username: username,
		password: password,
	}
}

func (m *basicAuthMiddleware) wrap(next ctxHandler) ctxHandler {
	return ctxHandlerFunc(func(c *context) {
		username, password, ok := c.BasicAuth()
		if !ok || username != m.username || password != m.password {
			http.Error(c.ResponseWriter, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.serveHTTP(c)
	})
}



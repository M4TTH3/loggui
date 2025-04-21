package server

import "net/http"

// This package provides a simple HTTP server to serve the static files
// and also handle client requests.

// Server is the main wrapper for all the loggui server functionality.
// It contains the HTTP handler and any other server related
//
// The server will use add the following endpoints: T.B.A.
type Server struct {
	username string
	password string

	http.Handler
}

func NewServer(username, password string) *Server {
	handler := newMux()
	s := &Server{
		username: username,
		password: password,
		Handler:  handler,
	}

	for _, m := range []middleware {
		newBasicAuthMiddleware(username, password),
	} {
		handler.use(m)
	}

	// Serve static files from the static directory
	fs := http.FileServer(http.Dir("static"))
	handler.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve the api endpoints

	return s
}

func (s *Server) ListenAndServe(addr string) error {
	if err := http.ListenAndServe(addr, s); err != nil {
		return err
	}

	return nil
}

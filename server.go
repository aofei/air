package air

import (
	"context"
	"net/http"
	"time"
)

// server represents the HTTP server.
type server struct {
	air    *Air
	server *http.Server
}

// newServer returns a new instance of the `server`.
func newServer(a *Air) *server {
	return &server{
		air:    a,
		server: &http.Server{},
	}
}

// serve starts the s.
func (s *server) serve() error {
	s.server.Addr = s.air.Address
	s.server.Handler = s
	s.server.ReadTimeout = s.air.ReadTimeout
	s.server.WriteTimeout = s.air.WriteTimeout
	s.server.MaxHeaderBytes = s.air.MaxHeaderBytes

	go func() {
		if err := s.air.minifier.init(); err != nil {
			s.air.Logger.Error(err)
		}
		if err := s.air.renderer.init(); err != nil {
			s.air.Logger.Error(err)
		}
		if err := s.air.coffer.init(); err != nil {
			s.air.Logger.Error(err)
		}
	}()

	if s.air.DebugMode {
		s.air.LoggerEnabled = true
		s.air.Logger.Debug("serving in debug mode")
	}

	if s.air.TLSCertFile != "" && s.air.TLSKeyFile != "" {
		return s.server.ListenAndServeTLS(
			s.air.TLSCertFile,
			s.air.TLSKeyFile,
		)
	}

	return s.server.ListenAndServe()
}

// close closes the s immediately.
func (s *server) close() error {
	return s.server.Close()
}

// shutdown gracefully shuts down the s without interrupting any active
// connections until timeout. It waits indefinitely for connections to return to
// idle and then shut down when the timeout is negative.
func (s *server) shutdown(timeout time.Duration) error {
	if timeout < 0 {
		return s.server.Shutdown(context.Background())
	}

	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.server.Shutdown(c)
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	req := newRequest(s.air, r)
	res := newResponse(req, rw)

	// Gases
	h := func(req *Request, res *Response) error {
		h := s.air.router.route(req)
		for i := len(s.air.Gases) - 1; i >= 0; i-- {
			h = s.air.Gases[i](h)
		}

		return h(req, res)
	}

	// Pregases
	for i := len(s.air.PreGases) - 1; i >= 0; i-- {
		h = s.air.PreGases[i](h)
	}

	// Execute chain
	if err := h(req, res); err != nil {
		s.air.ErrorHandler(err, req, res)
	}
}

package air

import (
	"context"
	"net/http"
	"time"
)

// server represents the HTTP server.
type server struct {
	server *http.Server
}

// serverSingleton is the singleton instance of the `server`.
var serverSingleton = &server{
	server: &http.Server{},
}

// serve starts the s.
func (s *server) serve() error {
	s.server.Addr = Address
	s.server.Handler = s
	s.server.ReadTimeout = ReadTimeout
	s.server.WriteTimeout = WriteTimeout
	s.server.MaxHeaderBytes = MaxHeaderBytes

	go func() {
		if err := rendererSingleton.init(); err != nil {
			ERROR(err)
		}
	}()

	if DebugMode {
		LoggerEnabled = true
		INFO("serving in debug mode")
	}

	if TLSCertFile != "" && TLSKeyFile != "" {
		return s.server.ListenAndServeTLS(TLSCertFile, TLSKeyFile)
	}

	return s.server.ListenAndServe()
}

// close closes the s immediately.
func (s *server) close() error {
	return s.server.Close()
}

// shutdown gracefully shuts down the s without interrupting any active
// connections until timeout. It waits indefinitely for connections to return to
// idle and then shut down when the timeout is less than or equal to 0.
func (s *server) shutdown(timeout time.Duration) error {
	if timeout <= 0 {
		return s.server.Shutdown(context.Background())
	}

	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.server.Shutdown(c)
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	req := newRequest(r)
	res := newResponse(req, rw)

	// Gases
	h := func(req *Request, res *Response) error {
		h := routerSingleton.route(req)
		for i := len(Gases) - 1; i >= 0; i-- {
			h = Gases[i](h)
		}
		return h(req, res)
	}

	// PreGases
	for i := len(PreGases) - 1; i >= 0; i-- {
		h = PreGases[i](h)
	}

	// Execute chain
	if err := h(req, res); err != nil {
		ErrorHandler(err, req, res)
		ERROR(err)
	}
}

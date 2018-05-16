package air

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

// server is an HTTP server.
type server struct {
	server *http.Server
}

// theServer is the singleton of the `server`.
var theServer = &server{
	server: &http.Server{},
}

// serve starts the s.
func (s *server) serve() error {
	s.server.Addr = Address
	s.server.Handler = s
	s.server.ReadTimeout = ReadTimeout
	s.server.ReadHeaderTimeout = ReadHeaderTimeout
	s.server.WriteTimeout = WriteTimeout
	s.server.IdleTimeout = IdleTimeout
	s.server.MaxHeaderBytes = MaxHeaderBytes

	if DebugMode {
		LoggerEnabled = true
		INFO("serving in debug mode")
	}

	if TLSCertFile != "" && TLSKeyFile != "" {
		if HTTPSEnforced && (Address == "" ||
			strings.HasSuffix(strings.ToLower(Address), ":https") ||
			strings.HasSuffix(Address, ":443")) {
			a, _, err := net.SplitHostPort(Address)
			if err != nil {
				a = Address
			}
			var h http.HandlerFunc
			h = func(rw http.ResponseWriter, r *http.Request) {
				host, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					host = r.Host
				}
				url := "https://" + host + r.RequestURI
				http.Redirect(rw, r, url, 301)
			}
			go func() {
				if err = http.ListenAndServe(a, h); err != nil {
					FATAL(err)
				}
			}()
		}
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
// idle and then shut down when the timeout is less than or equal to zero.
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
		h := theRouter.route(req)
		for i := len(Gases) - 1; i >= 0; i-- {
			h = Gases[i](h)
		}
		return h(req, res)
	}

	// Pregases
	for i := len(Pregases) - 1; i >= 0; i-- {
		h = Pregases[i](h)
	}

	// Execute chain
	if err := h(req, res); err != nil {
		ErrorHandler(err, req, res)
	}
}

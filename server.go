package air

import "net/http"

// server represents the HTTP server.
//
// It's embedded with the `http.Server`.
type server struct {
	*http.Server

	air *Air
}

// newServer returns a pointer of a new instance of the `server`.
func newServer(a *Air) *server {
	return &server{
		Server: &http.Server{},
		air:    a,
	}
}

// serve starts the HTTP server.
func (s *server) serve() error {
	c := s.air.Config

	s.Addr = c.Address
	s.Handler = s
	s.ReadTimeout = c.ReadTimeout
	s.WriteTimeout = c.WriteTimeout
	s.MaxHeaderBytes = c.MaxHeaderBytes

	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile)
	}

	return s.ListenAndServe()
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := s.air.contextPool.Get().(*Context)
	c.feed(req, rw)

	// Gases
	h := func(c *Context) error {
		if methodAllowed(c.Request.Method) {
			s.air.router.route(c.Request.Method, c.Request.URL.EscapedPath(), c)
		} else {
			c.Handler = MethodNotAllowedHandler
		}

		h := c.Handler
		for i := len(s.air.gases) - 1; i >= 0; i-- {
			h = s.air.gases[i](h)
		}

		return h(c)
	}

	// Pregases
	for i := len(s.air.pregases) - 1; i >= 0; i-- {
		h = s.air.pregases[i](h)
	}

	// Execute chain
	if err := h(c); err != nil {
		s.air.HTTPErrorHandler(err, c)
	}

	c.reset()
	s.air.contextPool.Put(c)
}

// methodAllowed reports whether the method is allowed.
func methodAllowed(method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

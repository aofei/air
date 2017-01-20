package air

import (
	"crypto/tls"
	"net/http"
)

// server represents the HTTP server.
//
// It's embedded with the `http.Server`.
type server struct {
	*http.Server

	air *Air
}

// newServer returns a pointer of a new instance of the `server`.
func newServer(a *Air) *server {
	s := &server{}

	s.Server = &http.Server{}
	s.Addr = a.Config.Address
	s.Handler = s
	s.ReadTimeout = a.Config.ReadTimeout
	s.WriteTimeout = a.Config.WriteTimeout

	if a.Config.DisableHTTP2 {
		s.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}

	s.air = a

	return s
}

// serve starts the HTTP server.
func (s *server) serve() error {
	cl := s.air.Config.Listener
	if cl != nil {
		return s.Serve(cl)
	}

	cert := s.air.Config.TLSCertFile
	key := s.air.Config.TLSKeyFile
	if cert != "" && key != "" {
		return s.ListenAndServeTLS(cert, key)
	}

	return s.ListenAndServe()
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := s.air.contextPool.Get().(*Context)
	c.feed(req, rw)

	// Gases
	h := func(c *Context) error {
		method := c.Request.Method
		path := c.Request.URL.EscapedPath()
		s.air.router.route(method, path, c)
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

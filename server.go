package air

import "github.com/valyala/fasthttp"

// server represents the HTTP server.
type server struct {
	fastServer *fasthttp.Server
	air        *Air
}

// newServer returns an new instance of `server`.
func newServer(a *Air) *server {
	s := &server{
		fastServer: &fasthttp.Server{},
		air:        a,
	}
	s.fastServer.ReadTimeout = s.air.Config.ReadTimeout
	s.fastServer.WriteTimeout = s.air.Config.WriteTimeout
	s.fastServer.Handler = s.serveHTTP
	return s
}

// start starts the HTTP server.
func (s *server) start() error {
	if s.air.Config.Listener == nil {
		return s.startDefaultListener()
	}
	return s.startCustomListener()

}

// startDefaultListener starts the default HTTP linsterner.
func (s *server) startDefaultListener() error {
	c := s.air.Config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.fastServer.ListenAndServeTLS(c.Address, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.fastServer.ListenAndServe(c.Address)
}

// startCustomListener starts the custom HTTP linsterner.
func (s *server) startCustomListener() error {
	c := s.air.Config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.fastServer.ServeTLS(c.Listener, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.fastServer.Serve(c.Listener)
}

// serveHTTP serves the HTTP requests.
func (s *server) serveHTTP(fastCtx *fasthttp.RequestCtx) {
	c := s.air.pool.context()

	// Request
	c.Request.fastCtx = fastCtx
	c.Request.Header.fastRequestHeader = &fastCtx.Request.Header
	c.Request.URI.fastURI = fastCtx.URI()

	// Response
	c.Response.fastCtx = fastCtx
	c.Response.Header.fastResponseHeader = &fastCtx.Response.Header
	c.Response.Writer = fastCtx

	// Gases
	h := func(*Context) error {
		method := c.Request.Method()
		path := c.Request.URI.PathOriginal()
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

	s.air.pool.put(c)
}

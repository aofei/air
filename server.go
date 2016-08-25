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
	s.fastServer.Handler = s.fastServeHTTP
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
func (s *server) serveHTTP(req *Request, res *Response) {
	c := s.air.pool.context()
	c.Request = req
	c.Response = res

	// Gases
	h := func(*Context) error {
		method := req.Method()
		path := req.URI.PathOriginal()
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

// fastServeHTTP serves the fast HTTP request.
func (s *server) fastServeHTTP(c *fasthttp.RequestCtx) {
	req := s.air.pool.request()
	reqHdr := s.air.pool.requestHeader()
	reqURI := s.air.pool.uri()

	res := s.air.pool.response()
	resHdr := s.air.pool.responseHeader()

	req.fastCtx = c
	req.Header = reqHdr
	req.URI = reqURI
	reqHdr.fastRequestHeader = &c.Request.Header
	reqURI.fastURI = c.URI()

	res.fastCtx = c
	res.Header = resHdr
	res.Writer = c
	resHdr.fastResponseHeader = &c.Response.Header

	s.serveHTTP(req, res)

	s.air.pool.put(req)
	s.air.pool.put(reqHdr)
	s.air.pool.put(reqURI)

	s.air.pool.put(res)
	s.air.pool.put(resHdr)
}

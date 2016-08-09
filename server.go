package air

import "github.com/valyala/fasthttp"

type (
	// server represents the HTTP server.
	server struct {
		fastServer *fasthttp.Server
		handler    serverHandler
		air        *Air
	}

	// serverHandler defines an interface to server HTTP requests via
	// `ServeHTTP(*Request, *Response)`.
	serverHandler interface {
		serveHTTP(*Request, *Response)
	}
)

// newServer returns an new instance of `server`.
func newServer(a *Air) *server {
	s := &server{
		fastServer: new(fasthttp.Server),
		handler:    a,
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

// fastServeHTTP serves the fast HTTP request.
func (s *server) fastServeHTTP(c *fasthttp.RequestCtx) {
	req := s.air.pool.request.Get().(*Request)
	reqHdr := s.air.pool.requestHeader.Get().(*RequestHeader)
	reqURI := s.air.pool.uri.Get().(*URI)

	res := s.air.pool.response.Get().(*Response)
	resHdr := s.air.pool.responseHeader.Get().(*ResponseHeader)

	req.fastCtx = c
	req.Header = reqHdr
	req.URI = reqURI
	reqHdr.fastRequestHeader = &c.Request.Header
	reqURI.fastURI = c.URI()

	res.fastCtx = c
	res.Header = resHdr
	res.Writer = c
	resHdr.fastResponseHeader = &c.Response.Header

	s.handler.serveHTTP(req, res)

	req.reset()
	reqHdr.reset()
	reqURI.reset()

	res.reset()
	resHdr.reset()

	s.air.pool.request.Put(req)
	s.air.pool.requestHeader.Put(reqHdr)
	s.air.pool.uri.Put(reqURI)

	s.air.pool.response.Put(res)
	s.air.pool.responseHeader.Put(resHdr)
}

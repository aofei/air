package air

import "github.com/valyala/fasthttp"

type (
	// Server represents the HTTP server.
	Server struct {
		fastServer *fasthttp.Server
		air        *Air

		Handler ServerHandler
	}

	// ServerHandler defines an interface to server HTTP requests via
	// `ServeHTTP(*Request, *Response)`.
	ServerHandler interface {
		ServeHTTP(*Request, *Response)
	}
)

// NewServer returns an new instance of `Server`.
func NewServer(a *Air) *Server {
	s := &Server{
		fastServer: new(fasthttp.Server),
		air:        a,
		Handler:    a,
	}
	s.fastServer.ReadTimeout = s.air.Config.ReadTimeout
	s.fastServer.WriteTimeout = s.air.Config.WriteTimeout
	s.fastServer.Handler = s.fastServeHTTP
	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	if s.air.Config.Listener == nil {
		return s.startDefaultListener()
	}
	return s.startCustomListener()

}

// startDefaultListener starts the default HTTP linsterner.
func (s *Server) startDefaultListener() error {
	c := s.air.Config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.fastServer.ListenAndServeTLS(c.Address, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.fastServer.ListenAndServe(c.Address)
}

// startCustomListener starts the custom HTTP linsterner.
func (s *Server) startCustomListener() error {
	c := s.air.Config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.fastServer.ServeTLS(c.Listener, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.fastServer.Serve(c.Listener)
}

// fastServeHTTP serves the fast HTTP request.
func (s *Server) fastServeHTTP(c *fasthttp.RequestCtx) {
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

	s.Handler.ServeHTTP(req, res)

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

// fastWrapHandler wraps `fasthttp.RequestHandler` into `HandlerFunc`.
func fastWrapHandler(h fasthttp.RequestHandler) HandlerFunc {
	return func(c *Context) error {
		req := c.Request
		res := c.Response
		ctx := req.fastCtx
		h(ctx)
		res.Status = ctx.Response.StatusCode()
		res.Size = int64(ctx.Response.Header.ContentLength())
		return nil
	}
}

// fastWrapGas wraps `func(fasthttp.RequestHandler) fasthttp.RequestHandler`
// into `GasFunc`
func fastWrapGas(m func(fasthttp.RequestHandler) fasthttp.RequestHandler) GasFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			req := c.Request
			res := c.Response
			ctx := req.fastCtx
			m(func(ctx *fasthttp.RequestCtx) {
				next(c)
			})(ctx)
			res.Status = ctx.Response.StatusCode()
			res.Size = int64(ctx.Response.Header.ContentLength())
			return
		}
	}
}

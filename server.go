package air

import (
	"sync"

	"github.com/valyala/fasthttp"
)

type (
	// Server represents the HTTP server.
	Server struct {
		fastServer *fasthttp.Server
		pool       *pool
		air        *Air

		Handler ServerHandler
		Logger  *Logger
	}

	// pool represents the pools of a HTTP server.
	pool struct {
		request        sync.Pool
		response       sync.Pool
		requestHeader  sync.Pool
		responseHeader sync.Pool
		uri            sync.Pool
	}

	// ServerHandler defines an interface to server HTTP requests via
	// `ServeHTTP(*Request, *Response)`.
	ServerHandler interface {
		ServeHTTP(*Request, *Response)
	}

	// serverHandlerFunc is an adapter to allow the use of `func(*Request, *Response)`
	// as an HTTP handler.
	serverHandlerFunc func(*Request, *Response)
)

// NewServer returns an new instance of `Server`.
func NewServer(a *Air) *Server {
	s := &Server{
		fastServer: new(fasthttp.Server),
		air:        a,
	}
	s.pool = &pool{
		request: sync.Pool{
			New: func() interface{} {
				return &Request{Logger: s.Logger}
			},
		},
		response: sync.Pool{
			New: func() interface{} {
				return &Response{Logger: s.Logger}
			},
		},
		requestHeader: sync.Pool{
			New: func() interface{} {
				return &RequestHeader{}
			},
		},
		responseHeader: sync.Pool{
			New: func() interface{} {
				return &ResponseHeader{}
			},
		},
		uri: sync.Pool{
			New: func() interface{} {
				return &URI{}
			},
		},
	}
	s.Handler = serverHandlerFunc(func(req *Request, res *Response) {
		s.Logger.Error("ServerHandler Not Set")
	})
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
	// Request
	req := s.pool.request.Get().(*Request)
	reqHdr := s.pool.requestHeader.Get().(*RequestHeader)
	reqURI := s.pool.uri.Get().(*URI)
	reqHdr.reset(&c.Request.Header)
	reqURI.reset(c.URI())
	req.reset(c, reqHdr, reqURI)

	// Response
	res := s.pool.response.Get().(*Response)
	resHdr := s.pool.responseHeader.Get().(*ResponseHeader)
	resHdr.reset(&c.Response.Header)
	res.reset(c, resHdr)

	s.Handler.ServeHTTP(req, res)

	// Return to pool
	s.pool.request.Put(req)
	s.pool.requestHeader.Put(reqHdr)
	s.pool.uri.Put(reqURI)
	s.pool.response.Put(res)
	s.pool.responseHeader.Put(resHdr)
}

// ServeHTTP implements `ServerHandler#ServeHTTP()`.
func (h serverHandlerFunc) ServeHTTP(req *Request, res *Response) {
	h(req, res)
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

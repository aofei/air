package air

import (
	"net"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type (
	// Server represents the HTTP server.
	Server struct {
		fastServer *fasthttp.Server
		config     *ServerConfig
		pool       *pool

		Handler ServerHandler
		Logger  Logger
	}

	// ServerConfig represents the HTTP server config.
	ServerConfig struct {
		Address      string        // TCP address to listen on.
		Listener     net.Listener  // Custom `net.Listener`. If set, server accepts connections on it.
		TLSCertFile  string        // TLS certificate file path.
		TLSKeyFile   string        // TLS key file path.
		ReadTimeout  time.Duration // Maximum duration before timing out read of the request.
		WriteTimeout time.Duration // Maximum duration before timing out write of the response.
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

// NewServer returns an new instance of `Server` with provided listen address.
func NewServer(addr string) *Server {
	c := &ServerConfig{Address: addr}
	return NewServerWithConfig(c)
}

// NewServerWithTLS returns an new instance of `Server` with provided TLS config.
func NewServerWithTLS(addr, certFile, keyFile string) *Server {
	c := &ServerConfig{
		Address:     addr,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}
	return NewServerWithConfig(c)
}

// NewServerWithConfig returns an new instance of `Server` with provided config.
func NewServerWithConfig(c *ServerConfig) *Server {
	s := &Server{
		fastServer: new(fasthttp.Server),
		config:     c,
		Logger:     NewLogger("air"),
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
	s.fastServer.ReadTimeout = c.ReadTimeout
	s.fastServer.WriteTimeout = c.WriteTimeout
	s.fastServer.Handler = s.fastServeHTTP
	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	if s.config.Listener == nil {
		return s.startDefaultListener()
	}
	return s.startCustomListener()

}

// startDefaultListener starts the default HTTP linsterner.
func (s *Server) startDefaultListener() error {
	c := s.config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.fastServer.ListenAndServeTLS(c.Address, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.fastServer.ListenAndServe(c.Address)
}

// startCustomListener starts the custom HTTP linsterner.
func (s *Server) startCustomListener() error {
	c := s.config
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

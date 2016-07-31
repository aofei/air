package air

import (
	"net"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type (
	// Server defines the interface for HTTP server.
	Server interface {
		// SetHandler sets the handler for the HTTP server.
		SetHandler(Handler)

		// SetLogger sets the logger for the HTTP server.
		SetLogger(Logger)

		// Start starts the HTTP server.
		Start() error
	}

	fastServer struct {
		*fasthttp.Server
		config  Config
		handler Handler
		logger  Logger
		pool    *pool
	}

	// Config defines fasthttp config.
	Config struct {
		Address      string        // TCP address to listen on.
		Listener     net.Listener  // Custom `net.Listener`. If set, server accepts connections on it.
		TLSCertFile  string        // TLS certificate file path.
		TLSKeyFile   string        // TLS key file path.
		ReadTimeout  time.Duration // Maximum duration before timing out read of the request.
		WriteTimeout time.Duration // Maximum duration before timing out write of the response.
	}

	// Handler defines an interface to server HTTP requests via `ServeHTTP(Request, Response)`
	// function.
	Handler interface {
		ServeHTTP(Request, Response)
	}

	// handlerFunc is an adapter to allow the use of `func(Request, Response)` as
	// an HTTP handler.
	handlerFunc func(Request, Response)

	pool struct {
		request        sync.Pool
		response       sync.Pool
		requestHeader  sync.Pool
		responseHeader sync.Pool
		uri            sync.Pool
	}
)

// NewServer returns `Server` with provided listen address.
func NewServer(addr string) Server {
	c := Config{Address: addr}
	return NewServerWithConfig(c)
}

// NewServerWithTLS returns `Server` with provided TLS config.
func NewServerWithTLS(addr, certFile, keyFile string) Server {
	c := Config{
		Address:     addr,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}
	return NewServerWithConfig(c)
}

// NewServerWithConfig returns `Server` with provided config.
func NewServerWithConfig(c Config) Server {
	s := &fastServer{
		Server: new(fasthttp.Server),
		config: c,
		logger: NewLogger("air"),
	}
	s.pool = &pool{
		request: sync.Pool{
			New: func() interface{} {
				return &Request{Logger: s.logger}
			},
		},
		response: sync.Pool{
			New: func() interface{} {
				return &Response{Logger: s.logger}
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
	s.handler = handlerFunc(func(req Request, res Response) {
		s.logger.Error("Handler Not Set, Use `SetHandler()` To Set It.")
	})
	s.ReadTimeout = c.ReadTimeout
	s.WriteTimeout = c.WriteTimeout
	s.Handler = s.ServeHTTP
	return s
}

func (s *fastServer) SetHandler(h Handler) {
	s.handler = h
}

func (s *fastServer) SetLogger(l Logger) {
	s.logger = l
}

func (s *fastServer) Start() error {
	if s.config.Listener == nil {
		return s.startDefaultListener()
	}
	return s.startCustomListener()

}

func (s *fastServer) startDefaultListener() error {
	c := s.config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.ListenAndServeTLS(c.Address, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.ListenAndServe(c.Address)
}

func (s *fastServer) startCustomListener() error {
	c := s.config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.ServeTLS(c.Listener, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.Serve(c.Listener)
}

func (s *fastServer) ServeHTTP(c *fasthttp.RequestCtx) {
	// Request
	req := s.pool.request.Get().(*Request)
	reqHdr := s.pool.requestHeader.Get().(*RequestHeader)
	reqURI := s.pool.uri.Get().(*URI)
	reqHdr.reset(&c.Request.Header)
	reqURI.reset(c.URI())
	req.reset(c, *reqHdr, *reqURI)

	// Response
	res := s.pool.response.Get().(*Response)
	resHdr := s.pool.responseHeader.Get().(*ResponseHeader)
	resHdr.reset(&c.Response.Header)
	res.reset(c, *resHdr)

	s.handler.ServeHTTP(*req, *res)

	// Return to pool
	s.pool.request.Put(req)
	s.pool.requestHeader.Put(reqHdr)
	s.pool.uri.Put(reqURI)
	s.pool.response.Put(res)
	s.pool.responseHeader.Put(resHdr)
}

// ServeHTTP serves HTTP request.
func (h handlerFunc) ServeHTTP(req Request, res Response) {
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

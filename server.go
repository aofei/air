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

	// FastServer implements `Server`.
	FastServer struct {
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

	// FastHandlerFunc is an adapter to allow the use of `func(Request, Response)` as
	// an HTTP handler.
	FastHandlerFunc func(Request, Response)

	pool struct {
		request        sync.Pool
		response       sync.Pool
		requestHeader  sync.Pool
		responseHeader sync.Pool
		uri            sync.Pool
	}
)

// ServeHTTP serves HTTP request.
func (h FastHandlerFunc) ServeHTTP(req Request, res Response) {
	h(req, res)
}

// NewServer returns `FastServer` with provided listen address.
func NewServer(addr string) *FastServer {
	c := Config{Address: addr}
	return WithConfig(c)
}

// WithTLS returns `FastServer` with provided TLS config.
func WithTLS(addr, certFile, keyFile string) *FastServer {
	c := Config{
		Address:     addr,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}
	return WithConfig(c)
}

// WithConfig returns `FastServer` with provided config.
func WithConfig(c Config) *FastServer {
	s := &FastServer{
		Server: new(fasthttp.Server),
		config: c,
		pool: &pool{
			request: sync.Pool{
				New: func() interface{} {
					return &FastRequest{logger: s.logger}
				},
			},
			response: sync.Pool{
				New: func() interface{} {
					return &FastResponse{logger: s.logger}
				},
			},
			requestHeader: sync.Pool{
				New: func() interface{} {
					return &FastRequestHeader{}
				},
			},
			responseHeader: sync.Pool{
				New: func() interface{} {
					return &FastResponseHeader{}
				},
			},
			uri: sync.Pool{
				New: func() interface{} {
					return &FastURI{}
				},
			},
		},
		handler: FastHandlerFunc(func(req Request, res Response) {
			s.logger.Error("handler not set, use `SetHandler()` to set it.")
		}),
		logger: NewLogger("air"),
	}
	s.ReadTimeout = c.ReadTimeout
	s.WriteTimeout = c.WriteTimeout
	s.Handler = s.ServeHTTP
	return
}

// SetHandler implements `Server#SetHandler` function.
func (s *FastServer) SetHandler(h Handler) {
	s.handler = h
}

// SetLogger implements `Server#SetLogger` function.
func (s *FastServer) SetLogger(l Logger) {
	s.logger = l
}

// Start implements `Server#Start` function.
func (s *FastServer) Start() error {
	if s.config.Listener == nil {
		return s.startDefaultListener()
	}
	return s.startCustomListener()

}

func (s *FastServer) startDefaultListener() error {
	c := s.config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.ListenAndServeTLS(c.Address, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.ListenAndServe(c.Address)
}

func (s *FastServer) startCustomListener() error {
	c := s.config
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.ServeTLS(c.Listener, c.TLSCertFile, c.TLSKeyFile)
	}
	return s.Serve(c.Listener)
}

func (s *FastServer) ServeHTTP(c *fasthttp.RequestCtx) {
	// Request
	req := s.pool.request.Get().(*FastRequest)
	reqHdr := s.pool.requestHeader.Get().(*FastRequestHeader)
	reqURI := s.pool.uri.Get().(*FastURI)
	reqHdr.reset(&c.Request.Header)
	reqURI.reset(c.URI())
	req.reset(c, reqHdr, reqURI)

	// Response
	res := s.pool.response.Get().(*FastResponse)
	resHdr := s.pool.responseHeader.Get().(*FastResponseHeader)
	resHdr.reset(&c.Response.Header)
	res.reset(c, resHdr)

	s.handler.ServeHTTP(req, res)

	// Return to pool
	s.pool.request.Put(req)
	s.pool.requestHeader.Put(reqHdr)
	s.pool.uri.Put(reqURI)
	s.pool.response.Put(res)
	s.pool.responseHeader.Put(resHdr)
}

// FastWrapHandler wraps `fasthttp.RequestHandler` into `HandlerFunc`.
func FastWrapHandler(h fasthttp.RequestHandler) HandlerFunc {
	return func(c Context) error {
		req := c.Request().(*FastRequest)
		res := c.Response().(*FastResponse)
		ctx := req.RequestCtx
		h(ctx)
		res.status = ctx.Response.StatusCode()
		res.size = int64(ctx.Response.Header.ContentLength())
		return nil
	}
}

// FastWrapGas wraps `func(fasthttp.RequestHandler) fasthttp.RequestHandler`
// into `GasFunc`
func FastWrapGas(m func(fasthttp.RequestHandler) fasthttp.RequestHandler) GasFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			req := c.Request().(*FastRequest)
			res := c.Response().(*FastResponse)
			ctx := req.RequestCtx
			m(func(ctx *fasthttp.RequestCtx) {
				next(c)
			})(ctx)
			res.status = ctx.Response.StatusCode()
			res.size = int64(ctx.Response.Header.ContentLength())
			return
		}
	}
}

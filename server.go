package air

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type (
	// Server represents the HTTP server.
	Server interface {
		// Precontain adds the gases to the chain which is perform
		// before the router.
		Precontain(gases ...Gas)

		// Contain adds the gases to the chain which is perform after
		// the router.
		Contain(gases ...Gas)

		// Serve starts the HTTP server.
		Serve() error

		// Close closes the HTTP server immediately.
		Close() error

		// Shutdown gracefully shuts down the HTTP server without
		// interrupting any active connections until timeout. It waits
		// indefinitely for connections to return to idle and then shut
		// down when the timeout is negative.
		Shutdown(timeout time.Duration) error

		// SetHTTPErrorHandler sets the heh to the centralized HTTP
		// error handler of the HTTP server.
		SetHTTPErrorHandler(heh HTTPErrorHandler)
	}

	// Gas defines a function to process gases.
	Gas func(Handler) Handler

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, *Context)

	// server implements the `Server`.
	server struct {
		air *Air

		pregases         []Gas
		gases            []Gas
		server           *http.Server
		contextPool      *sync.Pool
		httpErrorHandler HTTPErrorHandler
	}
)

// newServer returns a pointer of a new instance of the `server`.
func newServer(a *Air) *server {
	return &server{
		air:    a,
		server: &http.Server{},
		contextPool: &sync.Pool{
			New: func() interface{} {
				return NewContext(a)
			},
		},
		httpErrorHandler: func(err error, c *Context) {
			he := NewHTTPError(http.StatusInternalServerError)
			if che, ok := err.(*HTTPError); ok {
				he = che
			} else if c.Air.Config.DebugMode {
				he.Message = err.Error()
			}

			if !c.Response.Written {
				c.Response.StatusCode = he.Code
				c.String(he.Message)
			}

			c.Air.Logger.Error(err)
		},
	}
}

// Precontain implements the `Server#Precontain()`.
func (s *server) Precontain(gases ...Gas) {
	s.pregases = append(s.pregases, gases...)
}

// Contain implements the `Server#Contain()`.
func (s *server) Contain(gases ...Gas) {
	s.gases = append(s.gases, gases...)
}

// Serve implements the `Server#Serve()`.
func (s *server) Serve() error {
	c := s.air.Config

	s.server.Addr = c.Address
	s.server.Handler = s
	s.server.ReadTimeout = c.ReadTimeout
	s.server.WriteTimeout = c.WriteTimeout
	s.server.MaxHeaderBytes = c.MaxHeaderBytes

	if err := s.air.Minifier.Init(); err != nil {
		panic(err)
	} else if err := s.air.Renderer.Init(); err != nil {
		panic(err)
	} else if err := s.air.Coffer.Init(); err != nil {
		panic(err)
	}

	if c.DebugMode {
		c.LoggerEnabled = true
		s.air.Logger.Debug("serving in debug mode")
	}

	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.server.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile)
	}

	return s.server.ListenAndServe()
}

// Close implements the `Server#Close()`.
func (s *server) Close() error {
	return s.server.Close()
}

// Shutdown implements the `Server#Shutdown()`.
func (s *server) Shutdown(timeout time.Duration) error {
	if timeout < 0 {
		return s.server.Shutdown(context.Background())
	}

	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.server.Shutdown(c)
}

// SetHTTPErrorHandler implements the `Server#SetHTTPErrorHandler()`.
func (s *server) SetHTTPErrorHandler(heh HTTPErrorHandler) {
	s.httpErrorHandler = heh
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := s.contextPool.Get().(*Context)
	c.feed(req, rw)

	// Gases
	h := func(c *Context) error {
		s.air.router.route(
			c.Request.Method,
			c.Request.URL.EscapedPath(),
			c,
		)

		h := c.Handler
		for i := len(s.gases) - 1; i >= 0; i-- {
			h = s.gases[i](h)
		}

		return h(c)
	}

	// Pregases
	for i := len(s.pregases) - 1; i >= 0; i-- {
		h = s.pregases[i](h)
	}

	// Execute chain
	if err := h(c); err != nil {
		s.httpErrorHandler(err, c)
	}

	c.reset()
	s.contextPool.Put(c)
}

package air

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type (
	// Air is the top-level framework struct.
	Air struct {
		pregases    []Gas
		gases       []Gas
		paramCap    int
		contextPool *sync.Pool
		server      *http.Server
		router      *router

		Config           *Config
		Logger           Logger
		Binder           Binder
		Minifier         Minifier
		Renderer         Renderer
		Coffer           Coffer
		HTTPErrorHandler HTTPErrorHandler
	}

	// Handler defines a function to serve HTTP requests.
	Handler func(*Context) error

	// Gas defines a function to process gases.
	Gas func(Handler) Handler

	// HTTPError represents an error that occurred while handling an HTTP
	// request.
	HTTPError struct {
		Code    int
		Message string
	}

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, *Context)

	// Map is an alias of the `map[string]interface{}`.
	Map = map[string]interface{}
)

// HTTP error handlers
var (
	NotFoundHandler = func(c *Context) error {
		return NewHTTPError(http.StatusNotFound)
	}

	MethodNotAllowedHandler = func(c *Context) error {
		return NewHTTPError(http.StatusMethodNotAllowed)
	}
)

// New returns a pointer of a new instance of the `Air`.
func New() *Air {
	a := &Air{}

	a.contextPool = &sync.Pool{
		New: func() interface{} {
			return NewContext(a)
		},
	}
	a.server = &http.Server{}
	a.router = newRouter(a)

	a.Config = NewConfig("config.toml")
	a.Logger = newLogger(a)
	a.Binder = newBinder()
	a.Minifier = newMinifier()
	a.Renderer = newRenderer(a)
	a.Coffer = newCoffer(a)
	a.HTTPErrorHandler = DefaultHTTPErrorHandler

	return a
}

// Precontain adds the gases to the chain which is perform before the router.
func (a *Air) Precontain(gases ...Gas) {
	a.pregases = append(a.pregases, gases...)
}

// Contain adds the gases to the chain which is perform after the router.
func (a *Air) Contain(gases ...Gas) {
	a.gases = append(a.gases, gases...)
}

// GET registers a new GET route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.add("GET", path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) HEAD(path string, h Handler, gases ...Gas) {
	a.add("HEAD", path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.add("POST", path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.add("PUT", path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) PATCH(path string, h Handler, gases ...Gas) {
	a.add("PATCH", path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.add("DELETE", path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) CONNECT(path string, h Handler, gases ...Gas) {
	a.add("CONNECT", path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) OPTIONS(path string, h Handler, gases ...Gas) {
	a.add("OPTIONS", path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) TRACE(path string, h Handler, gases ...Gas) {
	a.add("TRACE", path, h, gases...)
}

// Static registers a new route with the path prefix to serve the static files
// from the provided root directory.
func (a *Air) Static(prefix, root string) {
	a.GET(prefix+"*", func(c *Context) error {
		err := c.File(path.Join(root, c.Param("*")))
		if os.IsNotExist(err) {
			return NotFoundHandler(c)
		}
		return err
	})
}

// File registers a new route with the path to serve a static file.
func (a *Air) File(path, file string) {
	a.GET(path, func(c *Context) error {
		err := c.File(file)
		if os.IsNotExist(err) {
			return NotFoundHandler(c)
		}
		return err
	})
}

// add registers a new route for the path with the method and the matching h in
// the router with the optional route-level gases.
func (a *Air) add(method, path string, h Handler, gases ...Gas) {
	a.router.add(method, path, func(c *Context) error {
		h := h
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}
		return h(c)
	})
}

// Serve starts the HTTP server.
func (a *Air) Serve() error {
	s := a.server
	c := a.Config

	s.Addr = c.Address
	s.Handler = a
	s.ReadTimeout = c.ReadTimeout
	s.WriteTimeout = c.WriteTimeout
	s.MaxHeaderBytes = c.MaxHeaderBytes

	if err := a.Minifier.Init(); err != nil {
		panic(err)
	} else if err := a.Renderer.Init(); err != nil {
		panic(err)
	} else if err := a.Coffer.Init(); err != nil {
		panic(err)
	}

	if c.DebugMode {
		c.LoggerEnabled = true
		a.Logger.Debug("serving in debug mode")
	}

	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return s.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile)
	}

	return s.ListenAndServe()
}

// Close closes the HTTP server immediately.
func (a *Air) Close() error {
	return a.server.Close()
}

// Shutdown gracefully shuts down the HTTP server without interrupting any
// active connections until timeout. It waits indefinitely for connections to
// return to idle and then shut down when the timeout is negative.
func (a *Air) Shutdown(timeout time.Duration) error {
	if timeout < 0 {
		return a.server.Shutdown(context.Background())
	}

	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return a.server.Shutdown(c)
}

// ServeHTTP implements the `http.Handler`.
func (a *Air) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := a.contextPool.Get().(*Context)
	c.feed(req, rw)

	// Gases
	h := func(c *Context) error {
		a.router.route(c.Request.Method, c.Request.URL.EscapedPath(), c)

		h := c.Handler
		for i := len(a.gases) - 1; i >= 0; i-- {
			h = a.gases[i](h)
		}

		return h(c)
	}

	// Pregases
	for i := len(a.pregases) - 1; i >= 0; i-- {
		h = a.pregases[i](h)
	}

	// Execute chain
	if err := h(c); err != nil {
		a.HTTPErrorHandler(err, c)
	}

	c.reset()
	a.contextPool.Put(c)
}

// WrapHandler wraps the h into the `Handler`.
func WrapHandler(h http.Handler) Handler {
	return func(c *Context) error {
		h.ServeHTTP(c.Response, c.Request.Request)
		return nil
	}
}

// WrapGas wraps the h into the `Gas`.
func WrapGas(h Handler) Gas {
	return func(next Handler) Handler {
		return func(c *Context) error {
			if err := h(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// NewHTTPError returns a pointer of a new instance of the `HTTPError`.
func NewHTTPError(code int, messages ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(messages) > 0 {
		he.Message = fmt.Sprint(messages...)
	}
	return he
}

// Error implements the `error#Error()`.
func (he *HTTPError) Error() string {
	return he.Message
}

// DefaultHTTPErrorHandler is the default HTTP error handler.
func DefaultHTTPErrorHandler(err error, c *Context) {
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
}

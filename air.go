package air

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"time"
)

type (
	// Air is the top-level framework struct.
	Air struct {
		paramCap int
		router   *router

		Config   *Config
		Logger   Logger
		Server   Server
		Binder   Binder
		Minifier Minifier
		Renderer Renderer
		Coffer   Coffer
	}

	// Handler defines a function to serve HTTP requests.
	Handler func(*Context) error

	// Map is an alias of the `map[string]interface{}`.
	Map = map[string]interface{}
)

// HTTP error handlers
var (
	NotFoundHandler = func(*Context) error {
		return NewHTTPError(http.StatusNotFound)
	}

	MethodNotAllowedHandler = func(*Context) error {
		return NewHTTPError(http.StatusMethodNotAllowed)
	}
)

// New returns a pointer of a new instance of the `Air`.
func New() *Air {
	a := &Air{}

	a.router = newRouter(a)

	a.Config = NewConfig("config.toml")
	a.Logger = newLogger(a)
	a.Server = newServer(a)
	a.Binder = newBinder()
	a.Minifier = newMinifier()
	a.Renderer = newRenderer(a)
	a.Coffer = newCoffer(a)

	return a
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

// Precontain is an alias for the `Server#Precontain()` of the a.
func (a *Air) Precontain(gases ...Gas) {
	a.Server.Precontain(gases...)
}

// Contain is an alias for the `Server#Contain()` of the a.
func (a *Air) Contain(gases ...Gas) {
	a.Server.Contain(gases...)
}

// Serve is an alias for the `Server#Serve()` of the a.
func (a *Air) Serve() error {
	return a.Server.Serve()
}

// Close is an alias for the `Server#CLose()` of the a.
func (a *Air) Close() error {
	return a.Server.Close()
}

// Shutdown is an alias for the `Server#Shutdown()` of the a.
func (a *Air) Shutdown(timeout time.Duration) error {
	return a.Server.Shutdown(timeout)
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

// HTTPError represents an error that occurred while handling an HTTP request.
type HTTPError struct {
	Code    int
	Message string
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

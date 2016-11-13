package air

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"runtime"
)

type (
	// Air is the top-level framework struct.
	Air struct {
		pregases []GasFunc
		gases    []GasFunc
		router   *router
		binder   *binder
		renderer *renderer

		Pool             *Pool
		Config           *Config
		Logger           *Logger
		HTTPErrorHandler HTTPErrorHandler
	}

	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(*Context) error

	// GasFunc defines a function to process gas.
	GasFunc func(HandlerFunc) HandlerFunc

	// HTTPError represents an error that occurred while handling a request.
	HTTPError struct {
		Code    int
		Message string
	}

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, *Context)

	// JSONMap is a map that stores data in JSON format.
	JSONMap map[string]interface{}
)

// HTTP methods (which follows the REST principle)
const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT" // The Air advises you to forget the PATCH.
	DELETE = "DELETE"
)

// For easy for-range
var methods = [4]string{GET, POST, PUT, DELETE}

// MIME types
const (
	MIMEApplicationJSON       = "application/json; charset=utf-8"
	MIMEApplicationJavaScript = "application/javascript; charset=utf-8"
	MIMEApplicationXML        = "application/xml; charset=utf-8"
	MIMEApplicationForm       = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf   = "application/protobuf"
	MIMEApplicationMsgpack    = "application/msgpack"
	MIMETextHTML              = "text/html; charset=utf-8"
	MIMETextPlain             = "text/plain; charset=utf-8"
	MIMEMultipartForm         = "multipart/form-data"
	MIMEOctetStream           = "application/octet-stream"
)

// Headers
const (
	HeaderAcceptEncoding                = "Accept-Encoding"
	HeaderAllow                         = "Allow"
	HeaderAuthorization                 = "Authorization"
	HeaderContentDisposition            = "Content-Disposition"
	HeaderContentEncoding               = "Content-Encoding"
	HeaderContentLength                 = "Content-Length"
	HeaderContentType                   = "Content-Type"
	HeaderCookie                        = "Cookie"
	HeaderSetCookie                     = "Set-Cookie"
	HeaderIfModifiedSince               = "If-Modified-Since"
	HeaderLastModified                  = "Last-Modified"
	HeaderLocation                      = "Location"
	HeaderUpgrade                       = "Upgrade"
	HeaderVary                          = "Vary"
	HeaderWWWAuthenticate               = "WWW-Authenticate"
	HeaderXForwardedProto               = "X-Forwarded-Proto"
	HeaderXHTTPMethodOverride           = "X-HTTP-Method-Override"
	HeaderXForwardedFor                 = "X-Forwarded-For"
	HeaderXRealIP                       = "X-Real-IP"
	HeaderServer                        = "Server"
	HeaderOrigin                        = "Origin"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

// Errors
var (
	ErrUnauthorized                = NewHTTPError(http.StatusUnauthorized)          // 401
	ErrNotFound                    = NewHTTPError(http.StatusNotFound)              // 404
	ErrMethodNotAllowed            = NewHTTPError(http.StatusMethodNotAllowed)      // 405
	ErrStatusRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge) // 413
	ErrUnsupportedMediaType        = NewHTTPError(http.StatusUnsupportedMediaType)  // 415

	ErrInternalServerError = NewHTTPError(http.StatusInternalServerError) // 500
	ErrBadGateway          = NewHTTPError(http.StatusBadGateway)          // 502
	ErrServiceUnavailable  = NewHTTPError(http.StatusServiceUnavailable)  // 503
	ErrGatewayTimeout      = NewHTTPError(http.StatusGatewayTimeout)      // 504

	ErrInvalidRedirectCode = errors.New("invalid redirect status code")
	ErrCookieNotFound      = errors.New("cookie not found")
)

// HTTP error handlers
var (
	NotFoundHandler = func(c *Context) error {
		return ErrNotFound
	}

	MethodNotAllowedHandler = func(c *Context) error {
		return ErrMethodNotAllowed
	}
)

// New returns a new instance of `Air`.
func New() *Air {
	a := &Air{}
	a.router = newRouter(a)
	a.binder = newBinder(a)
	a.renderer = newRenderer(a)
	a.Pool = newPool(a)
	a.Config = newConfig()
	a.Logger = newLogger(a)
	a.HTTPErrorHandler = defaultHTTPErrorHandler
	return a
}

// Precontain adds gases to the chain which is run before router.
func (a *Air) Precontain(gases ...GasFunc) {
	a.pregases = append(a.pregases, gases...)
}

// Contain adds gases to the chain which is run after router.
func (a *Air) Contain(gases ...GasFunc) {
	a.gases = append(a.gases, gases...)
}

// Get registers a new GET route for a path with matching handler in the router
// with optional route-level gases.
func (a *Air) Get(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(GET, path, handler, gases...)
}

// Post registers a new POST route for a path with matching handler in the
// router with optional route-level gases.
func (a *Air) Post(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(POST, path, handler, gases...)
}

// Put registers a new PUT route for a path with matching handler in the
// router with optional route-level gases.
func (a *Air) Put(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(PUT, path, handler, gases...)
}

// Delete registers a new DELETE route for a path with matching handler in
// the router with optional route-level gases.
func (a *Air) Delete(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(DELETE, path, handler, gases...)
}

// Static registers a new route with path prefix to serve static files from
// the provided root directory.
func (a *Air) Static(prefix, root string) {
	a.Get(prefix+"*", func(c *Context) error {
		return c.File(path.Join(root, c.Params[c.ParamNames[0]]))
	})
}

// File registers a new route with path to serve a static file.
func (a *Air) File(path, file string) {
	a.Get(path, func(c *Context) error {
		return c.File(file)
	})
}

// add registers a new route for a path with a HTTP method and matching handler
// in the router with optional route-level gases.
func (a *Air) add(method, path string, handler HandlerFunc, gases ...GasFunc) {
	name := handlerName(handler)
	a.router.add(method, path, func(c *Context) error {
		h := handler
		// Chain gases
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}
		return h(c)
	})
	r := &route{
		method:  method,
		path:    path,
		handler: name,
	}
	a.router.routes[method+path] = r
}

// URI returns a URI generated from handler with optional params.
func (a *Air) URI(handler HandlerFunc, params ...interface{}) string {
	uri := &bytes.Buffer{}
	ln := len(params)
	n := 0
	name := handlerName(handler)
	for _, r := range a.router.routes {
		if r.handler == name {
			for i, l := 0, len(r.path); i < l; i++ {
				if r.path[i] == ':' && n < ln {
					for ; i < l && r.path[i] != '/'; i++ {
					}
					uri.WriteString(fmt.Sprintf("%v", params[n]))
					n++
				}
				if i < l {
					uri.WriteByte(r.path[i])
				}
			}
			break
		}
	}
	return uri.String()
}

// SetTemplateFunc sets the f into template func map with a name.
func (a *Air) SetTemplateFunc(name string, f interface{}) {
	a.renderer.templateFuncMap[name] = f
}

// Run starts the HTTP server.
func (a *Air) Run() {
	a.renderer.parseTemplates()
	if a.Config.DebugMode {
		a.Logger.Level = DEBUG
		a.Logger.Debug("running in debug mode")
	}

	s := newServer(a)
	if err := s.start(); err != nil {
		panic(err)
	}
}

// handlerName returns the handler's func name.
func handlerName(handler HandlerFunc) string {
	t := reflect.ValueOf(handler).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	}
	return t.String()
}

// WrapGas wraps `HandlerFunc` into `GasFunc`.
func WrapGas(handler HandlerFunc) GasFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if err := handler(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// NewHTTPError returns a new instance of `HTTPError`.
func NewHTTPError(code int, msg ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(msg) > 0 {
		he.Message = fmt.Sprint(msg...)
	}
	return he
}

// Error implements `Error#Error()`.
func (he *HTTPError) Error() string {
	return he.Message
}

// defaultHTTPErrorHandler invokes the default HTTP error handler.
func defaultHTTPErrorHandler(err error, c *Context) {
	he := ErrInternalServerError
	if che, ok := err.(*HTTPError); ok {
		he = che
	}
	if c.Air.Config.DebugMode {
		he.Message = err.Error()
	}
	if !c.Response.Committed {
		c.Data["string"] = he.Message
		c.StatusCode = he.Code
		c.String()
	}
	c.Air.Logger.Error(err)
}

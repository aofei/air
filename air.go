package air

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"sync"

	"golang.org/x/net/context"
)

type (
	// Air is the top-level framework struct.
	Air struct {
		pregases         []GasFunc
		gases            []GasFunc
		maxParam         *int
		notFoundHandler  HandlerFunc
		httpErrorHandler HTTPErrorHandler
		binder           Binder
		renderer         Renderer
		pool             sync.Pool
		debug            bool
		router           *Router
		logger           Logger
	}

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method  string
		Path    string
		Handler string
	}

	// HTTPError represents an error that occurred while handling a request.
	HTTPError struct {
		Code    int
		Message string
	}

	// GasFunc defines a function to process gas.
	GasFunc func(HandlerFunc) HandlerFunc

	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(Context) error

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, Context)

	// Validator is the interface that wraps the Validate function.
	Validator interface {
		Validate() error
	}

	// Renderer is the interface that wraps the Render function.
	Renderer interface {
		Render(io.Writer, string, interface{}, Context) error
	}
)

// HTTP methods (which follows the REST principles)
const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT" // The Air advise you to forget the PATCH.
	DELETE = "DELETE"
)

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

// For easy for-range
var methods = [4]string{GET, POST, PUT, DELETE}

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

	ErrRendererNotRegistered = errors.New("Renderer Not Registered")
	ErrInvalidRedirectCode   = errors.New("Invalid Redirect Status Code")
	ErrCookieNotFound        = errors.New("Cookie Not Found")
)

// Error handlers
var (
	NotFoundHandler = func(c Context) error {
		return ErrNotFound
	}

	MethodNotAllowedHandler = func(c Context) error {
		return ErrMethodNotAllowed
	}
)

// New creates an instance of Air.
func New() *Air {
	a := &Air{maxParam: new(int)}
	a.pool.New = func() interface{} {
		return a.NewContext(nil, nil)
	}
	a.router = NewRouter(a)

	// Defaults
	a.SetHTTPErrorHandler(a.DefaultHTTPErrorHandler)
	a.SetBinder(&airBinder{})
	l := NewLogger("air")
	l.SetLevel(ERROR)
	a.SetLogger(l)
	return a
}

// NewContext returns a Context instance.
func (a *Air) NewContext(req Request, res Response) Context {
	return &airContext{
		context:  context.Background(),
		request:  req,
		response: res,
		air:      a,
		pvalues:  make([]string, *a.maxParam),
		handler:  NotFoundHandler,
	}
}

// Router returns router.
func (a *Air) Router() *Router {
	return a.router
}

// Logger returns the logger instance.
func (a *Air) Logger() Logger {
	return a.logger
}

// SetLogger defines a custom logger.
func (a *Air) SetLogger(l Logger) {
	a.logger = l
}

// SetLogOutput sets the output destination for the logger. Default value is `os.Std*`
func (a *Air) SetLogOutput(w io.Writer) {
	a.logger.SetOutput(w)
}

// SetLogLevel sets the log level for the logger. Default value ERROR.
func (a *Air) SetLogLevel(l LoggerLevel) {
	a.logger.SetLevel(l)
}

// DefaultHTTPErrorHandler invokes the default HTTP error handler.
func (a *Air) DefaultHTTPErrorHandler(err error, c Context) {
	code := http.StatusInternalServerError
	msg := http.StatusText(code)
	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
	}
	if a.debug {
		msg = err.Error()
	}
	if !c.Response().Committed() {
		c.String(code, msg)
	}
	a.logger.Error(err)
}

// SetHTTPErrorHandler registers a custom Air.HTTPErrorHandler.
func (a *Air) SetHTTPErrorHandler(h HTTPErrorHandler) {
	a.httpErrorHandler = h
}

// Binder returns the binder instance.
func (a *Air) Binder() Binder {
	return a.binder
}

// SetBinder registers a custom binder. It's invoked by `Context#Bind()`.
func (a *Air) SetBinder(b Binder) {
	a.binder = b
}

// SetRenderer registers an HTML template renderer. It's invoked by `Context#Render()`.
func (a *Air) SetRenderer(r Renderer) {
	a.renderer = r
}

// Debug returns debug mode (enabled or disabled).
func (a *Air) Debug() bool {
	return a.debug
}

// SetDebug enables/disables debug mode.
func (a *Air) SetDebug(on bool) {
	a.debug = on
}

// Precontain adds gases to the chain which is run before router.
func (a *Air) Precontain(gases ...GasFunc) {
	a.pregases = append(a.pregases, gases...)
}

// Contain adds gases to the chain which is run after router.
func (a *Air) Contain(gases ...GasFunc) {
	a.gases = append(a.gases, gases...)
}

// GET registers a new GET route for a path with matching handler in the router
// with optional route-level gases.
func (a *Air) GET(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(GET, path, handler, gases...)
}

// POST registers a new POST route for a path with matching handler in the
// router with optional route-level gases.
func (a *Air) POST(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(POST, path, handler, gases...)
}

// PUT registers a new PUT route for a path with matching handler in the
// router with optional route-level gases.
func (a *Air) PUT(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(PUT, path, handler, gases...)
}

// DELETE registers a new DELETE route for a path with matching handler in the router
// with optional route-level gases.
func (a *Air) DELETE(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(DELETE, path, handler, gases...)
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (a *Air) Static(prefix, root string) {
	a.GET(prefix+"*", func(c Context) error {
		return c.File(path.Join(root, c.P(0)))
	})
}

// File registers a new route with path to serve a static file.
func (a *Air) File(path, file string) {
	a.GET(path, func(c Context) error {
		return c.File(file)
	})
}

func (a *Air) add(method, path string, handler HandlerFunc, gases ...GasFunc) {
	name := handlerName(handler)
	a.router.Add(method, path, func(c Context) error {
		h := handler
		// Chain gases
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}
		return h(c)
	}, a)
	r := Route{
		Method:  method,
		Path:    path,
		Handler: name,
	}
	a.router.routes[method+path] = r
}

// Group creates a new router group with prefix and optional group-level gases.
func (a *Air) Group(prefix string, gases ...GasFunc) *Group {
	g := &Group{prefix: prefix, air: a}
	g.Contain(gases...)
	return g
}

// URI generates a URI from handler.
func (a *Air) URI(handler HandlerFunc, params ...interface{}) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0
	name := handlerName(handler)
	for _, r := range a.router.routes {
		if r.Handler == name {
			for i, l := 0, len(r.Path); i < l; i++ {
				if r.Path[i] == ':' && n < ln {
					for ; i < l && r.Path[i] != '/'; i++ {
					}
					uri.WriteString(fmt.Sprintf("%v", params[n]))
					n++
				}
				if i < l {
					uri.WriteByte(r.Path[i])
				}
			}
			break
		}
	}
	return uri.String()
}

// Routes returns the registered routes.
func (a *Air) Routes() []Route {
	routes := []Route{}
	for _, v := range a.router.routes {
		routes = append(routes, v)
	}
	return routes
}

// AcquireContext returns an empty `Context` instance from the pool.
// You must be return the context by calling `ReleaseContext()`.
func (a *Air) AcquireContext() Context {
	return a.pool.Get().(Context)
}

// ReleaseContext returns the `Context` instance back to the pool.
// You must call it after `AcquireContext()`.
func (a *Air) ReleaseContext(c Context) {
	a.pool.Put(c)
}

func (a *Air) ServeHTTP(req Request, res Response) {
	c := a.pool.Get().(*airContext)
	c.Reset(req, res)

	// Gases
	h := func(Context) error {
		method := req.Method()
		path := req.URI().Path()
		a.router.Find(method, path, c)
		h := c.handler
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
		a.httpErrorHandler(err, c)
	}

	a.pool.Put(c)
}

// Run starts the HTTP server.
func (a *Air) Run(addr string) {
	s := NewServer(addr)
	s.SetHandler(a)
	s.SetLogger(a.logger)
	if a.Debug() {
		a.SetLogLevel(DEBUG)
		a.logger.Debug("Running In Debug Mode")
	}
	a.logger.Error(s.Start())
}

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, msg ...string) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(msg) > 0 {
		he.Message = msg[0]
	}
	return he
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	return he.Message
}

// WrapGas wrap `HandlerFunc` into `GasFunc`.
func WrapGas(handler HandlerFunc) GasFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if err := handler(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

func handlerName(handler HandlerFunc) string {
	t := reflect.ValueOf(handler).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	}
	return t.String()
}

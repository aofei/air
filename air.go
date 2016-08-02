package air

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"sync"

	"golang.org/x/net/context"
)

type (
	// Air is the top-level framework type.
	Air struct {
		pregases         []GasFunc
		gases            []GasFunc
		maxParam         *int
		pool             sync.Pool
		NotFoundHandler  HandlerFunc
		HTTPErrorHandler HTTPErrorHandler
		Binder           *Binder
		Renderer         *Renderer
		Debug            bool
		Router           *Router
		Logger           Logger
	}

	// HTTPError represents an error that occurred while handling a request.
	HTTPError struct {
		Code    int
		Message string
	}

	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(*Context) error

	// GasFunc defines a function to process gas.
	GasFunc func(HandlerFunc) HandlerFunc

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, *Context)
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

	ErrInvalidRedirectCode = errors.New("Invalid Redirect Status Code")
	ErrCookieNotFound      = errors.New("Cookie Not Found")

	ErrDataTemplateNotSetted = errors.New("c.Data[\"template\"] Not Setted")
	ErrDataHTMLNotSetted     = errors.New("c.Data[\"html\"] Not Setted")
	ErrDataStringNotSetted   = errors.New("c.Data[\"string\"] Not Setted")
	ErrDataJSONNotSetted     = errors.New("c.Data[\"json\"] Not Setted")
	ErrDataJSONPNotSetted    = errors.New("c.Data[\"jsonp\"] Not Setted")
	ErrDataCallbackNotSetted = errors.New("c.Data[\"callback\"] Not Setted")
	ErrDataXMLNotSetted      = errors.New("c.Data[\"xml\"] Not Setted")
)

// Error handlers
var (
	NotFoundHandler = func(c *Context) error {
		return ErrNotFound
	}

	MethodNotAllowedHandler = func(c *Context) error {
		return ErrMethodNotAllowed
	}
)

// New creates an instance of Air.
func New() *Air {
	a := &Air{maxParam: new(int)}
	a.pool.New = func() interface{} {
		return a.NewContext(&Request{}, &Response{})
	}
	a.Router = NewRouter(a)

	// Defaults
	a.HTTPErrorHandler = a.DefaultHTTPErrorHandler
	a.Binder = &Binder{}
	a.Renderer = &Renderer{}
	a.Renderer.initDefaultTempleFuncMap()
	l := NewLogger("air")
	l.SetLevel(ERROR)
	a.Logger = l
	return a
}

// NewContext returns a Context instance.
func (a *Air) NewContext(req *Request, res *Response) *Context {
	return &Context{
		goContext:   context.Background(),
		Request:     req,
		Response:    res,
		Air:         a,
		ParamValues: make([]string, *a.maxParam),
		Data:        make(map[string]interface{}),
		StatusCode:  http.StatusOK,
		Handler:     NotFoundHandler,
	}
}

// DefaultHTTPErrorHandler invokes the default HTTP error handler.
func (a *Air) DefaultHTTPErrorHandler(err error, c *Context) {
	code := http.StatusInternalServerError
	msg := http.StatusText(code)
	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
	}
	if a.Debug {
		msg = err.Error()
	}
	if !c.Response.Committed {
		c.Data["string"] = msg
		c.StatusCode = code
		c.String()
	}
	a.Logger.Error(err)
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

// Delete registers a new DELETE route for a path with matching handler in the router
// with optional route-level gases.
func (a *Air) Delete(path string, handler HandlerFunc, gases ...GasFunc) {
	a.add(DELETE, path, handler, gases...)
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (a *Air) Static(prefix, root string) {
	a.Get(prefix+"*", func(c *Context) error {
		return c.File(path.Join(root, c.P(0)))
	})
}

// File registers a new route with path to serve a static file.
func (a *Air) File(path, file string) {
	a.Get(path, func(c *Context) error {
		return c.File(file)
	})
}

func (a *Air) add(method, path string, handler HandlerFunc, gases ...GasFunc) {
	name := handlerName(handler)
	a.Router.Add(method, path, func(c *Context) error {
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
	a.Router.Routes[method+path] = r
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
	for _, r := range a.Router.Routes {
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

func (a *Air) ServeHTTP(req *Request, res *Response) {
	c := a.pool.Get().(*Context)
	c.Reset(req, res)

	// Gases
	h := func(*Context) error {
		method := req.Method()
		path := req.URI.Path()
		a.Router.Find(method, path, c)
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

	a.pool.Put(c)
}

// Run starts the HTTP server.
func (a *Air) Run(addr string) {
	s := NewServer(addr)
	s.SetHandler(a)
	s.SetLogger(a.Logger)
	if a.Debug {
		a.Logger.SetLevel(DEBUG)
		a.Logger.Debug("Running In Debug Mode")
	}
	a.Renderer.parseTemplates("views")
	a.Logger.Error(s.Start())
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
		return func(c *Context) error {
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

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
)

type (
	// Air is the top-level framework struct.
	Air struct {
		pool     *pool
		pregases []GasFunc
		gases    []GasFunc
		router   *router
		binder   *binder
		renderer *renderer

		Config           *Config
		HTTPErrorHandler HTTPErrorHandler
		Logger           *Logger
	}

	// pool represents the pools of `Air`.
	pool struct {
		context        *sync.Pool
		request        *sync.Pool
		response       *sync.Pool
		requestHeader  *sync.Pool
		responseHeader *sync.Pool
		uri            *sync.Pool
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
	a.pool = newPool(a)
	a.router = newRouter(a)
	a.binder = newBinder(a)
	a.renderer = newRenderer(a)
	a.Config = NewConfig("air")
	a.HTTPErrorHandler = a.defaultHTTPErrorHandler
	a.Logger = NewLogger(a)
	a.Logger.Level = ERROR
	return a
}

// defaultHTTPErrorHandler invokes the default HTTP error handler.
func (a *Air) defaultHTTPErrorHandler(err error, c *Context) {
	code := http.StatusInternalServerError
	msg := http.StatusText(code)
	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
	}
	if a.Config.DebugMode {
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
	}, a)
	r := route{
		method:  method,
		path:    path,
		handler: name,
	}
	a.router.routes[method+path] = r
}

// SetTemplateFunc sets the f into template func map with a name.
func (a *Air) SetTemplateFunc(name string, f interface{}) {
	a.renderer.templateFuncMap[name] = f
}

// URI returns a URI generated from handler with optional params.
func (a *Air) URI(handler HandlerFunc, params ...interface{}) string {
	uri := new(bytes.Buffer)
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

// Run starts the HTTP server.
func (a *Air) Run() {
	a.renderer.parseTemplates()
	if a.Config.DebugMode {
		a.Logger.Level = DEBUG
		a.Logger.Debug("Running In Debug Mode")
	}
	s := newServer(a)
	a.Logger.Error(s.start())
}

// serveHTTP implements `serverHandler#serveHTTP()`.
func (a *Air) serveHTTP(req *Request, res *Response) {
	c := a.pool.context.Get().(*Context)
	c.Request = req
	c.Response = res

	// Gases
	h := func(*Context) error {
		method := req.Method()
		path := req.URI.Path()
		a.router.find(method, path, c)
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
	a.pool.context.Put(c)
}

// newPool returnes a new instance of `pool`.
func newPool(a *Air) *pool {
	return &pool{
		context: &sync.Pool{
			New: func() interface{} {
				return newContext(a)
			},
		},
		request: &sync.Pool{
			New: func() interface{} {
				return newRequest(a)
			},
		},
		response: &sync.Pool{
			New: func() interface{} {
				return newResponse(a)
			},
		},
		requestHeader: &sync.Pool{
			New: func() interface{} {
				return &RequestHeader{}
			},
		},
		responseHeader: &sync.Pool{
			New: func() interface{} {
				return &ResponseHeader{}
			},
		},
		uri: &sync.Pool{
			New: func() interface{} {
				return &URI{}
			},
		},
	}
}

// NewHTTPError returns a new instance of `HTTPError`.
func NewHTTPError(code int, msg ...string) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(msg) > 0 {
		he.Message = msg[0]
	}
	return he
}

// Error implements `error#Error()`.
func (he *HTTPError) Error() string {
	return he.Message
}

// handlerName returns the handler's func name.
func handlerName(handler HandlerFunc) string {
	t := reflect.ValueOf(handler).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	}
	return t.String()
}

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
		pregases    []Gas
		gases       []Gas
		paramCap    int
		contextPool *sync.Pool
		server      *server
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

	// HTTPError represents an error that occurred while handling an HTTP request.
	HTTPError struct {
		Code    int
		Message string
	}

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, *Context)

	// Map is an alias of the `map[string]interface{}`.
	Map = map[string]interface{}
)

// MIME types
const (
	MIMEApplicationJSON               = "application/json"
	MIMEApplicationJavaScript         = "application/javascript"
	MIMEApplicationXML                = "application/xml"
	MIMEApplicationXWWWFormURLEncoded = "application/x-www-form-urlencoded"
	MIMEImageJPEG                     = "image/jpeg"
	MIMEImagePNG                      = "image/png"
	MIMEImageSVGXML                   = "image/svg+xml"
	MIMEMultipartFormData             = "multipart/form-data"
	MIMETextCSS                       = "text/css"
	MIMETextHTML                      = "text/html"
	MIMETextJavaScript                = "text/javascript"
	MIMETextPlain                     = "text/plain"
	MIMETextXML                       = "text/xml"

	CharsetUTF8 = "; charset=utf-8"
)

// HTTP methods
const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)

// For easy for-range
var methods = [4]string{GET, POST, PUT, DELETE}

// HTTP headers
const (
	HeaderAccept                          = "Accept"
	HeaderAcceptCharset                   = "Accept-Charset"
	HeaderAcceptEncoding                  = "Accept-Encoding"
	HeaderAcceptLanguage                  = "Accept-Language"
	HeaderAcceptRanges                    = "Accept-Ranges"
	HeaderAccessControlAllowCredentials   = "Access-Control-Allow-Credentials"
	HeaderAccessControlAllowHeaders       = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowMethods       = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowOrigin        = "Access-Control-Allow-Origin"
	HeaderAccessControlExposeHeaders      = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge             = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders     = "Access-Control-Request-Headers"
	HeaderAccessControlRequestMethod      = "Access-Control-Request-Method"
	HeaderAge                             = "Age"
	HeaderAllow                           = "Allow"
	HeaderAuthorization                   = "Authorization"
	HeaderCacheControl                    = "Cache-Control"
	HeaderConnection                      = "Connection"
	HeaderContentDisposition              = "Content-Disposition"
	HeaderContentEncoding                 = "Content-Encoding"
	HeaderContentLanguage                 = "Content-Language"
	HeaderContentLength                   = "Content-Length"
	HeaderContentLocation                 = "Content-Location"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderContentType                     = "Content-Type"
	HeaderCookie                          = "Cookie"
	HeaderDNT                             = "DNT"
	HeaderDate                            = "Date"
	HeaderETag                            = "ETag"
	HeaderExpires                         = "Expires"
	HeaderForm                            = "Form"
	HeaderHost                            = "Host"
	HeaderIfMatch                         = "If-Match"
	HeaderIfModifiedSince                 = "If-Modified-Since"
	HeaderIfNoneMatch                     = "If-None-Match"
	HeaderIfRange                         = "If-Range"
	HeaderIfUnmodifiedSince               = "If-Unmodified-Since"
	HeaderKeepAlive                       = "Keep-Alive"
	HeaderLastModified                    = "Last-Modified"
	HeaderLocation                        = "Location"
	HeaderOrigin                          = "Origin"
	HeaderPublicKeyPins                   = "Public-Key-Pins"
	HeaderPublicKeyPinsReportOnly         = "Public-Key-Pins-Report-Only"
	HeaderReferer                         = "Referer"
	HeaderReferrerPolicy                  = "Referrer-Policy"
	HeaderRetryAfter                      = "Retry-After"
	HeaderServer                          = "Server"
	HeaderSetCookie                       = "Set-Cookie"
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderTE                              = "TE"
	HeaderTK                              = "TK"
	HeaderTrailer                         = "Trailer"
	HeaderTransferEncoding                = "Transfer-Encoding"
	HeaderUpgrade                         = "Upgrade"
	HeaderUpgradeInsecureRequests         = "Upgrade-Insecure-Requests"
	HeaderUserAgent                       = "User-Agent"
	HeaderVary                            = "Vary"
	HeaderVia                             = "Via"
	HeaderWWWAuthenticate                 = "WWW-Authenticate"
	HeaderWarning                         = "Warning"
	HeaderXCSRFToken                      = "X-CSRF-Token"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXDNSPrefetchControl             = "X-DNS-Prefetch-Control"
	HeaderXForwardedFor                   = "X-Forwarded-For"
	HeaderXForwardedProto                 = "X-Forwarded-Proto"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderXHTTPMethodOverride             = "X-HTTP-Method-Override"
	HeaderXRealIP                         = "X-Real-IP"
	HeaderXXSSProtection                  = "X-XSS-Protection"
)

// HTTP errors
var (
	ErrUnauthorized          = NewHTTPError(http.StatusUnauthorized)          // 401
	ErrNotFound              = NewHTTPError(http.StatusNotFound)              // 404
	ErrMethodNotAllowed      = NewHTTPError(http.StatusMethodNotAllowed)      // 405
	ErrRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge) // 413
	ErrUnsupportedMediaType  = NewHTTPError(http.StatusUnsupportedMediaType)  // 415

	ErrInternalServerError = NewHTTPError(http.StatusInternalServerError) // 500
	ErrBadGateway          = NewHTTPError(http.StatusBadGateway)          // 502
	ErrServiceUnavailable  = NewHTTPError(http.StatusServiceUnavailable)  // 503
	ErrGatewayTimeout      = NewHTTPError(http.StatusGatewayTimeout)      // 504

	ErrInvalidRedirectCode = errors.New("invalid redirect status code")
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

// New returns a pointer of a new instance of the `Air`.
func New() *Air {
	a := &Air{}

	a.contextPool = &sync.Pool{
		New: func() interface{} {
			return NewContext(a)
		},
	}
	a.server = newServer(a)
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

// GET registers a new GET route for the path with the matching h in the router with the optional
// route-level gases.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.add(GET, path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the router with the optional
// route-level gases.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.add(POST, path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router with the optional
// route-level gases.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.add(PUT, path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the router with the
// optional route-level gases.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.add(DELETE, path, h, gases...)
}

// Static registers a new route with the path prefix to serve the static files from the provided
// root directory.
func (a *Air) Static(prefix, root string) {
	a.GET(prefix+"*", func(c *Context) error {
		return c.File(path.Join(root, c.Param("*")))
	})
}

// File registers a new route with the path to serve a static file.
func (a *Air) File(path, file string) {
	a.GET(path, func(c *Context) error {
		return c.File(file)
	})
}

// add registers a new route for the path with the method and the matching h in the router with the
// optional route-level gases.
func (a *Air) add(method, path string, h Handler, gases ...Gas) {
	hn := handlerName(h)

	a.router.add(method, path, func(c *Context) error {
		h := h
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}
		return h(c)
	})

	a.router.routes[method+path] = &route{
		method:  method,
		path:    path,
		handler: hn,
	}
}

// URL returns an URL generated from the h with the optional params.
func (a *Air) URL(h Handler, params ...interface{}) string {
	url := &bytes.Buffer{}
	hn := handlerName(h)
	ln := len(params)
	n := 0

	for _, r := range a.router.routes {
		if r.handler == hn {
			for i, l := 0, len(r.path); i < l; i++ {
				if r.path[i] == ':' && n < ln {
					for ; i < l && r.path[i] != '/'; i++ {
					}
					url.WriteString(fmt.Sprintf("%v", params[n]))
					n++
				}

				if i < l {
					url.WriteByte(r.path[i])
				}
			}

			break
		}
	}

	return url.String()
}

// Serve starts the HTTP server.
func (a *Air) Serve() error {
	if a.Config.DebugMode {
		a.Config.LoggerEnabled = true
		a.Logger.Debug("serving in debug mode")
	}

	go func() {
		if err := a.Minifier.Init(); err != nil {
			a.Logger.Error(err)
		}

		if err := a.Renderer.Init(); err != nil {
			a.Logger.Error(err)
		}

		if err := a.Coffer.Init(); err != nil {
			a.Logger.Error(err)
		}
	}()

	return a.server.serve()
}

// Close closes the HTTP server immediately.
func (a *Air) Close() error {
	return a.server.Close()
}

// Shutdown gracefully shuts down the HTTP server without interrupting any active connections.
func (a *Air) Shutdown(c *Context) error {
	return a.server.Shutdown(c.Context)
}

// handlerName returns the func name of the h.
func handlerName(h Handler) string {
	return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
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
	he := ErrInternalServerError
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

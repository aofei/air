package air

import (
	"fmt"
	"net/http"
	"path"
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

// HTTP methods
const (
	GET     = "GET"
	HEAD    = "HEAD"
	POST    = "POST"
	PUT     = "PUT"
	PATCH   = "PATCH"
	DELETE  = "DELETE"
	CONNECT = "CONNECT"
	OPTIONS = "OPTIONS"
	TRACE   = "TRACE"
)

// methods is an array containing all the HTTP methods for easy for-range.
var methods = [...]string{
	GET,
	HEAD,
	POST,
	PUT,
	PATCH,
	DELETE,
	CONNECT,
	OPTIONS,
	TRACE,
}

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
	ErrBadRequest                    = NewHTTPError(http.StatusBadRequest)                    // 400
	ErrUnauthorized                  = NewHTTPError(http.StatusUnauthorized)                  // 401
	ErrPaymentRequired               = NewHTTPError(http.StatusPaymentRequired)               // 402
	ErrForbidden                     = NewHTTPError(http.StatusForbidden)                     // 403
	ErrNotFound                      = NewHTTPError(http.StatusNotFound)                      // 404
	ErrMethodNotAllowed              = NewHTTPError(http.StatusMethodNotAllowed)              // 405
	ErrNotAcceptable                 = NewHTTPError(http.StatusNotAcceptable)                 // 406
	ErrProxyAuthRequired             = NewHTTPError(http.StatusProxyAuthRequired)             // 407
	ErrRequestTimeout                = NewHTTPError(http.StatusRequestTimeout)                // 408
	ErrConflict                      = NewHTTPError(http.StatusConflict)                      // 409
	ErrGone                          = NewHTTPError(http.StatusGone)                          // 410
	ErrLengthRequired                = NewHTTPError(http.StatusLengthRequired)                // 411
	ErrPreconditionFailed            = NewHTTPError(http.StatusPreconditionFailed)            // 412
	ErrRequestEntityTooLarge         = NewHTTPError(http.StatusRequestEntityTooLarge)         // 413
	ErrRequestURITooLong             = NewHTTPError(http.StatusRequestURITooLong)             // 414
	ErrUnsupportedMediaType          = NewHTTPError(http.StatusUnsupportedMediaType)          // 415
	ErrRequestedRangeNotSatisfiable  = NewHTTPError(http.StatusRequestedRangeNotSatisfiable)  // 416
	ErrExpectationFailed             = NewHTTPError(http.StatusExpectationFailed)             // 417
	ErrTeapot                        = NewHTTPError(http.StatusTeapot)                        // 418
	ErrUnprocessableEntity           = NewHTTPError(http.StatusUnprocessableEntity)           // 422
	ErrLocked                        = NewHTTPError(http.StatusLocked)                        // 423
	ErrFailedDependency              = NewHTTPError(http.StatusFailedDependency)              // 424
	ErrUpgradeRequired               = NewHTTPError(http.StatusUpgradeRequired)               // 426
	ErrPreconditionRequired          = NewHTTPError(http.StatusPreconditionRequired)          // 428
	ErrTooManyRequests               = NewHTTPError(http.StatusTooManyRequests)               // 429
	ErrRequestHeaderFieldsTooLarge   = NewHTTPError(http.StatusRequestHeaderFieldsTooLarge)   // 431
	ErrUnavailableForLegalReasons    = NewHTTPError(http.StatusUnavailableForLegalReasons)    // 451
	ErrInternalServerError           = NewHTTPError(http.StatusInternalServerError)           // 500
	ErrNotImplemented                = NewHTTPError(http.StatusNotImplemented)                // 501
	ErrBadGateway                    = NewHTTPError(http.StatusBadGateway)                    // 502
	ErrServiceUnavailable            = NewHTTPError(http.StatusServiceUnavailable)            // 503
	ErrGatewayTimeout                = NewHTTPError(http.StatusGatewayTimeout)                // 504
	ErrHTTPVersionNotSupported       = NewHTTPError(http.StatusHTTPVersionNotSupported)       // 505
	ErrVariantAlsoNegotiates         = NewHTTPError(http.StatusVariantAlsoNegotiates)         // 506
	ErrInsufficientStorage           = NewHTTPError(http.StatusInsufficientStorage)           // 507
	ErrLoopDetected                  = NewHTTPError(http.StatusLoopDetected)                  // 508
	ErrNotExtended                   = NewHTTPError(http.StatusNotExtended)                   // 510
	ErrNetworkAuthenticationRequired = NewHTTPError(http.StatusNetworkAuthenticationRequired) // 511
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

// GET registers a new GET route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.add(GET, path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) HEAD(path string, h Handler, gases ...Gas) {
	a.add(HEAD, path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.add(POST, path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.add(PUT, path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) PATCH(path string, h Handler, gases ...Gas) {
	a.add(PATCH, path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.add(DELETE, path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) CONNECT(path string, h Handler, gases ...Gas) {
	a.add(CONNECT, path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) OPTIONS(path string, h Handler, gases ...Gas) {
	a.add(OPTIONS, path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) TRACE(path string, h Handler, gases ...Gas) {
	a.add(TRACE, path, h, gases...)
}

// Static registers a new route with the path prefix to serve the static files
// from the provided root directory.
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

// Shutdown gracefully shuts down the HTTP server without interrupting any
// active connections.
func (a *Air) Shutdown(c *Context) error {
	return a.server.Shutdown(c)
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

package air

import (
	"compress/gzip"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/BurntSushi/toml"
)

// Air is the top-level struct of this framework.
type Air struct {
	// AppName is the name of the current web application.
	//
	// The default value is "air".
	AppName string `toml:"app_name"`

	// MaintainerEmail is the e-mail address of the one who is responsible
	// for maintaining the current web application.
	//
	// The default value is "".
	MaintainerEmail string `toml:"maintainer_email"`

	// DebugMode indicates whether the current web application is in debug
	// mode.
	//
	// ATTENTION: Some features will be affected in debug mode.
	//
	// The default value is false.
	DebugMode bool `toml:"debug_mode"`

	// Address is the TCP address that the server listens on.
	//
	// The default value is ":8080".
	Address string `toml:"address"`

	// HostWhitelist is the hosts allowed by the server.
	//
	// It only works when the `DebugMode` is false.
	//
	// The default value is nil.
	HostWhitelist []string `toml:"host_whitelist"`

	// ReadTimeout is the maximum duration the server reads the request.
	//
	// The default value is 0.
	ReadTimeout time.Duration `toml:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed the server reads the
	// request headers.
	//
	// The default value is 0.
	ReadHeaderTimeout time.Duration `toml:"read_header_timeout"`

	// WriteTimeout is the maximum duration the server writes an response.
	//
	// The default value is 0.
	WriteTimeout time.Duration `toml:"write_timeout"`

	// IdleTimeout is the maximum amount of time the server waits for the
	// next request. If it is zero, the value of `ReadTimeout` is used. If
	// both are zero, the value `ReadHeaderTimeout` is used.
	//
	// The default value is 0.
	IdleTimeout time.Duration `toml:"idle_timeout"`

	// MaxHeaderBytes is the maximum number of bytes the server will read
	// parsing the request header's names and values, including the request
	// line.
	//
	// The default value is 1048576.
	MaxHeaderBytes int `toml:"max_header_bytes"`

	// TLSCertFile is the path to the TLS certificate file used when
	// starting the server.
	//
	// The default value is "".
	TLSCertFile string `toml:"tls_cert_file"`

	// TLSKeyFile is the path to the TLS key file used when starting the
	// server.
	//
	// The default value is "".
	TLSKeyFile string `toml:"tls_key_file"`

	// ACMEEnabled indicates whether the ACME is enabled.
	//
	// It only works when the `DebugMode` is false and both of the
	// `TLSCertFile` and the `TLSKeyFile` are empty.
	//
	// The default value is false.
	ACMEEnabled bool `toml:"acme_enabled"`

	// ACMECertRoot is the root of the ACME certificates.
	//
	// The default value is "acme-certs".
	ACMECertRoot string `toml:"acme_cert_root"`

	// HTTPSEnforced indicates whether the HTTPS is enforced.
	//
	// The default value is false.
	HTTPSEnforced bool `toml:"https_enforced"`

	// WebSocketHandshakeTimeout is the maximum amount of time the server
	// waits for the WebSocket handshake to complete.
	//
	// The default value is 0.
	WebSocketHandshakeTimeout time.Duration `toml:"websocket_handshake_timeout"`

	// WebSocketSubprotocols is the supported WebSocket subprotocols of the
	// server.
	//
	// The default value is nil.
	WebSocketSubprotocols []string `toml:"websocket_subprotocols"`

	// NotFoundHandler is a `Handler` that returns not found error.
	//
	// The default value is the `DefaultNotFoundHandler`.
	NotFoundHandler func(*Request, *Response) error `toml:"-"`

	// MethodNotAllowedHandler is a `Handler` that returns method not
	// allowed error.
	//
	// The default value is the `DefaultMethodNotAllowedHandler`.
	MethodNotAllowedHandler func(*Request, *Response) error `toml:"-"`

	// ErrorHandler is the centralized error handler of the server.
	//
	// The default value is the `DefaultErrorHandler`.
	ErrorHandler func(error, *Request, *Response) `toml:"-"`

	// ErrorLogger is the `log.Logger` that logs errors that occur in the
	// server.
	//
	// If nil, logging is done via the log package's standard logger.
	//
	// The default value is nil.
	ErrorLogger *log.Logger `toml:"-"`

	// Pregases is the `Gas` chain stack that performs before routing.
	//
	// The default value is nil.
	Pregases []Gas `toml:"-"`

	// Gases is the `Gas` chain stack that performs after routing.
	//
	// The default value is nil.
	Gases []Gas `toml:"-"`

	// AutoPushEnabled indicates whether the auto push is enabled.
	//
	// The default value is false.
	AutoPushEnabled bool `toml:"auto_push_enabled"`

	// MinifierEnabled indicates whether the minifier is enabled.
	//
	// The default value is false.
	MinifierEnabled bool `toml:"minifier_enabled"`

	// MinifierMIMETypes is the MIME types that will be minified.
	// Unsupported MIME types will be silently ignored.
	//
	// The default value is ["text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "image/svg+xml"].
	MinifierMIMETypes []string `toml:"minifier_mime_types"`

	// GzipEnabled indicates whether the gzip is enabled.
	//
	// The default value is false.
	GzipEnabled bool `toml:"gzip_enabled"`

	// GzipCompressionLevel is the compression level of the gzip.
	//
	// The default value is `gzip.DefaultCompression`.
	GzipCompressionLevel int `toml:"gzip_compression_level"`

	// GzipMIMETypes is the MIME types that will be gzipped.
	//
	// The default value is ["text/plain", "text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "image/svg+xml"].
	GzipMIMETypes []string `toml:"gzip_mime_types"`

	// TemplateRoot is the root of the HTML templates. All the HTML
	// templates inside it will be recursively parsed into the renderer.
	//
	// The default value is "templates".
	TemplateRoot string `toml:"template_root"`

	// TemplateExts is the filename extensions of the HTML templates used to
	// distinguish the HTML template files in the `TemplateRoot` when
	// parsing them into the renderer.
	//
	// The default value is [".html"].
	TemplateExts []string `toml:"template_exts"`

	// TemplateLeftDelim is the left side of the HTML template delimiter the
	// renderer renders the HTML templates.
	//
	// The default value is "{{".
	TemplateLeftDelim string `toml:"template_left_delim"`

	// TemplateRightDelim is the right side of the HTML template delimiter
	// the renderer renders the HTML templates.
	//
	// The default value is "}}".
	TemplateRightDelim string `toml:"template_right_delim"`

	// TemplateFuncMap is the HTML template function map the renderer
	// renders the HTML templates.
	//
	// The default value contains strlen, substr and timefmt.
	TemplateFuncMap template.FuncMap `toml:"-"`

	// CofferEnabled indicates whether the coffer is enabled.
	//
	// The default value is false.
	CofferEnabled bool `toml:"coffer_enabled"`

	// CofferMaxMemoryBytes is the maximum number of bytes of the runtime
	// memory the coffer will use.
	//
	// The default value is 33554432.
	CofferMaxMemoryBytes int `toml:"coffer_max_memory_bytes"`

	// AssetRoot is the root of the asset files. All the asset files inside
	// it will be recursively parsed into the coffer.
	//
	// The default value is "assets".
	AssetRoot string `toml:"asset_root"`

	// AssetExts is the filename extensions of the asset files used to
	// distinguish the asset files in the `AssetRoot` when loading them into
	// the coffer.
	//
	// The default value is [".html", ".css", ".js", ".json", ".xml",
	// ".svg", ".jpg", ".jpeg", ".png", ".gif"].
	AssetExts []string `toml:"asset_exts"`

	// I18nEnabled indicates whether the i18n is enabled.
	//
	// The default value is false.
	I18nEnabled bool `toml:"i18n_enabled"`

	// LocaleRoot is the root of the locale files. All the locale files
	// inside it will be parsed into the i18n.
	//
	// The default value is "locales".
	LocaleRoot string `toml:"locale_root"`

	// LocaleBase is the base of the locale files. It will be used when a
	// locale file cannot be found.
	//
	// The default value is "en-US".
	LocaleBase string `toml:"locale_base"`

	// ConfigFile is the TOML-based configuration file that will be parsed
	// into the matching fields before starting the server.
	//
	// The default value is "".
	ConfigFile string `toml:"-"`

	errorLogger                  *log.Logger
	server                       *server
	router                       *router
	binder                       *binder
	minifier                     *minifier
	renderer                     *renderer
	coffer                       *coffer
	i18n                         *i18n
	contentTypeSnifferBufferPool *sync.Pool
	reverseProxyTransport        *http.Transport
	reverseProxyBufferPool       *reverseProxyBufferPool
}

// Default is the default instance of the `Air`.
var Default = New()

// New returns a new instance of the `Air`.
func New() *Air {
	a := &Air{
		AppName:                 "air",
		Address:                 ":8080",
		MaxHeaderBytes:          1 << 20,
		ACMECertRoot:            "acme-certs",
		NotFoundHandler:         DefaultNotFoundHandler,
		MethodNotAllowedHandler: DefaultMethodNotAllowedHandler,
		ErrorHandler:            DefaultErrorHandler,
		MinifierMIMETypes: []string{
			"text/html",
			"text/css",
			"application/javascript",
			"application/json",
			"application/xml",
			"image/svg+xml",
		},
		GzipCompressionLevel: gzip.DefaultCompression,
		GzipMIMETypes: []string{
			"text/plain",
			"text/html",
			"text/css",
			"application/javascript",
			"application/json",
			"application/xml",
			"image/svg+xml",
		},
		TemplateRoot:       "templates",
		TemplateExts:       []string{".html"},
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		TemplateFuncMap: template.FuncMap{
			"strlen":  strlen,
			"substr":  substr,
			"timefmt": timefmt,
		},
		CofferMaxMemoryBytes: 32 << 20,
		AssetRoot:            "assets",
		AssetExts: []string{
			".html",
			".css",
			".js",
			".json",
			".xml",
			".svg",
			".jpg",
			".jpeg",
			".png",
			".gif",
		},
		LocaleRoot: "locales",
		LocaleBase: "en-US",
	}

	a.errorLogger = log.New(newErrorLogWriter(a), "air: ", 0)
	a.server = newServer(a)
	a.router = newRouter(a)
	a.binder = newBinder(a)
	a.minifier = newMinifier(a)
	a.renderer = newRenderer(a)
	a.coffer = newCoffer(a)
	a.i18n = newI18n(a)
	a.contentTypeSnifferBufferPool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 512)
		},
	}

	a.reverseProxyTransport = newReverseProxyTransport()
	a.reverseProxyBufferPool = newReverseProxyBufferPool()

	return a
}

// GET registers a new GET route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodGet, path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) HEAD(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodHead, path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPost, path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPut, path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) PATCH(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPatch, path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodDelete, path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) CONNECT(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodConnect, path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) OPTIONS(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodOptions, path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) TRACE(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodTrace, path, h, gases...)
}

// BATCH registers a batch of routes for the methods and the path with the
// matching h in the router with the optional route-level gases.
//
// The methods must either be nil (means all) or consists of one or more of the
// "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS" and
// "TRACE". Invalid methods will be silently dropped.
func (a *Air) BATCH(methods []string, path string, h Handler, gases ...Gas) {
	if methods == nil {
		methods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
			http.MethodTrace,
		}
	}

	for _, m := range methods {
		switch m {
		case http.MethodGet:
			a.GET(path, h, gases...)
		case http.MethodHead:
			a.HEAD(path, h, gases...)
		case http.MethodPost:
			a.POST(path, h, gases...)
		case http.MethodPut:
			a.PUT(path, h, gases...)
		case http.MethodPatch:
			a.PATCH(path, h, gases...)
		case http.MethodDelete:
			a.DELETE(path, h, gases...)
		case http.MethodConnect:
			a.CONNECT(path, h, gases...)
		case http.MethodOptions:
			a.OPTIONS(path, h, gases...)
		case http.MethodTrace:
			a.TRACE(path, h, gases...)
		}
	}
}

// FILE registers a new GET route and a new HEAD route with the path to serve a
// static file with the filename and the optional route-level gases.
func (a *Air) FILE(path, filename string, gases ...Gas) {
	h := func(req *Request, res *Response) error {
		err := res.WriteFile(filename)
		if os.IsNotExist(err) {
			return a.NotFoundHandler(req, res)
		}

		return err
	}

	a.BATCH([]string{http.MethodGet, http.MethodHead}, path, h, gases...)
}

// FILES registers a new GET route and a new HEAD route with the path prefix to
// serve the static files from the root with the optional route-level gases.
func (a *Air) FILES(prefix, root string, gases ...Gas) {
	if strings.HasSuffix(prefix, "/") {
		prefix += "*"
	} else {
		prefix += "/*"
	}

	if root == "" {
		root = "."
	}

	h := func(req *Request, res *Response) error {
		p := req.Param("*")
		if p == nil {
			return a.NotFoundHandler(req, res)
		}

		path := p.Value().String()
		path = filepath.FromSlash("/" + path)
		path = filepath.Clean(path)

		err := res.WriteFile(filepath.Join(root, path))
		if os.IsNotExist(err) {
			return a.NotFoundHandler(req, res)
		}

		return err
	}

	a.BATCH([]string{http.MethodGet, http.MethodHead}, prefix, h, gases...)
}

// Group returns a new instance of the `Group` with the path prefix and the
// optional group-level gases.
func (a *Air) Group(prefix string, gases ...Gas) *Group {
	return &Group{
		Air:    a,
		Prefix: prefix,
		Gases:  gases,
	}
}

// Serve starts the server.
func (a *Air) Serve() error {
	if a.ConfigFile != "" {
		if _, err := toml.DecodeFile(a.ConfigFile, a); err != nil {
			return err
		}
	}

	return a.server.serve()
}

// Close closes the server immediately.
func (a *Air) Close() error {
	return a.server.close()
}

// Shutdown gracefully shuts down the server without interrupting any active
// connections until timeout. It waits indefinitely for connections to return to
// idle and then shut down when the timeout is less than or equal to zero.
func (a *Air) Shutdown(timeout time.Duration) error {
	return a.server.shutdown(timeout)
}

// Handler defines a function to serve requests.
type Handler func(*Request, *Response) error

// DefaultNotFoundHandler is the default `Handler` that returns not found error.
func DefaultNotFoundHandler(req *Request, res *Response) error {
	res.Status = http.StatusNotFound
	return errors.New(http.StatusText(res.Status))
}

// DefaultMethodNotAllowedHandler is the default `Handler` that returns method
// not allowed error.
func DefaultMethodNotAllowedHandler(req *Request, res *Response) error {
	res.Status = http.StatusMethodNotAllowed
	return errors.New(http.StatusText(res.Status))
}

// DefaultErrorHandler is the default centralized error handler for the server.
func DefaultErrorHandler(err error, req *Request, res *Response) {
	if res.ContentLength > 0 {
		return
	}

	m := err.Error()
	if !req.Air.DebugMode && res.Status == http.StatusInternalServerError {
		m = http.StatusText(res.Status)
	}

	res.WriteString(m)
}

// Gas defines a function to process gases.
type Gas func(Handler) Handler

// WrapHTTPMiddleware provides a convenient way to wrap an `http.Handler`
// middleware into a `Gas`.
func WrapHTTPMiddleware(hm func(http.Handler) http.Handler) Gas {
	return func(next Handler) Handler {
		return func(req *Request, res *Response) error {
			var err error
			hm(http.HandlerFunc(func(
				rw http.ResponseWriter,
				r *http.Request,
			) {
				req.SetHTTPRequest(r)
				res.SetHTTPResponseWriter(rw)
				err = next(req, res)
			})).ServeHTTP(
				res.HTTPResponseWriter(),
				req.HTTPRequest(),
			)

			return err
		}
	}
}

// errorLogWriter is an error log writer.
type errorLogWriter struct {
	a *Air
}

// newErrorLogWriter returns a new instance of the `errorLogWriter` with the a.
func newErrorLogWriter(a *Air) *errorLogWriter {
	return &errorLogWriter{
		a: a,
	}
}

// Write implements the `io.Writer`.
func (elw *errorLogWriter) Write(b []byte) (int, error) {
	s := *(*string)(unsafe.Pointer(&b))
	if elw.a.ErrorLogger != nil {
		return len(s), elw.a.ErrorLogger.Output(2, s)
	}

	return len(s), log.Output(2, s)
}

// stringSliceContains reports whether the ss contains the s.
func stringSliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}

	return false
}

// stringSliceContainsCIly reports whether the ss contains the s
// case-insensitively.
func stringSliceContainsCIly(ss []string, s string) bool {
	s = strings.ToLower(s)
	for _, v := range ss {
		if strings.ToLower(v) == s {
			return true
		}
	}

	return false
}

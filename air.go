package air

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/BurntSushi/toml"
	ini "gopkg.in/ini.v1"
	yaml "gopkg.in/yaml.v2"
)

// Air is the top-level struct of this framework.
type Air struct {
	// AppName is the name of the current web application.
	//
	// The default value is "air".
	AppName string `json:"app_name" xml:"app_name" toml:"app_name" yaml:"app_name" ini:"app_name"`

	// MaintainerEmail is the e-mail address of the one who is responsible
	// for maintaining the current web application.
	//
	// The default value is "".
	MaintainerEmail string `json:"maintainer_email" xml:"maintainer_email" toml:"maintainer_email" yaml:"maintainer_email" ini:"maintainer_email"`

	// DebugMode indicates whether the current web application is in debug
	// mode.
	//
	// ATTENTION: Some features will be affected in debug mode.
	//
	// The default value is false.
	DebugMode bool `json:"debug_mode" xml:"debug_mode" toml:"debug_mode" yaml:"debug_mode" ini:"debug_mode"`

	// Address is the TCP address that the server listens on.
	//
	// The default value is ":8080".
	Address string `json:"address" xml:"address" toml:"address" yaml:"address" ini:"address"`

	// HostWhitelist is the hosts allowed by the server.
	//
	// It only works when the `DebugMode` is false.
	//
	// The default value is nil.
	HostWhitelist []string `json:"host_whitelist" xml:"host_whitelist" toml:"host_whitelist" yaml:"host_whitelist" ini:"host_whitelist"`

	// ReadTimeout is the maximum duration the server reads the request.
	//
	// The default value is 0.
	ReadTimeout time.Duration `json:"read_timeout" xml:"read_timeout" toml:"read_timeout" yaml:"read_timeout" ini:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed the server reads the
	// request headers.
	//
	// The default value is 0.
	ReadHeaderTimeout time.Duration `json:"read_header_timeout" xml:"read_header_timeout" toml:"read_header_timeout" yaml:"read_header_timeout" ini:"read_header_timeout"`

	// WriteTimeout is the maximum duration the server writes an response.
	//
	// The default value is 0.
	WriteTimeout time.Duration `json:"write_timeout" xml:"write_timeout" toml:"write_timeout" yaml:"write_timeout" ini:"write_timeout"`

	// IdleTimeout is the maximum amount of time the server waits for the
	// next request. If it is zero, the value of `ReadTimeout` is used. If
	// both are zero, the value `ReadHeaderTimeout` is used.
	//
	// The default value is 0.
	IdleTimeout time.Duration `json:"idle_timeout" xml:"idle_timeout" toml:"idle_timeout" yaml:"idle_timeout" ini:"idle_timeout"`

	// MaxHeaderBytes is the maximum number of bytes the server will read
	// parsing the request header's names and values, including the request
	// line.
	//
	// The default value is 1048576.
	MaxHeaderBytes int `json:"max_header_bytes" xml:"max_header_bytes" toml:"max_header_bytes" yaml:"max_header_bytes" ini:"max_header_bytes"`

	// TLSCertFile is the path to the TLS certificate file used when
	// starting the server.
	//
	// The default value is "".
	TLSCertFile string `json:"tls_cert_file" xml:"tls_cert_file" toml:"tls_cert_file" yaml:"tls_cert_file" ini:"tls_cert_file"`

	// TLSKeyFile is the path to the TLS key file used when starting the
	// server.
	//
	// The default value is "".
	TLSKeyFile string `json:"tls_key_file" xml:"tls_key_file" toml:"tls_key_file" yaml:"tls_key_file" ini:"tls_key_file"`

	// ACMEEnabled indicates whether the ACME is enabled.
	//
	// It only works when the `DebugMode` is false and both of the
	// `TLSCertFile` and the `TLSKeyFile` are empty.
	//
	// The default value is false.
	ACMEEnabled bool `json:"acme_enabled" xml:"acme_enabled" toml:"acme_enabled" yaml:"acme_enabled" ini:"acme_enabled"`

	// ACMECertRoot is the root of the ACME certificates.
	//
	// The default value is "acme-certs".
	ACMECertRoot string `json:"acme_cert_root" xml:"acme_cert_root" toml:"acme_cert_root" yaml:"acme_cert_root" ini:"acme_cert_root"`

	// HTTPSEnforced indicates whether the HTTPS is enforced.
	//
	// The default value is false.
	HTTPSEnforced bool `json:"https_enforced" xml:"https_enforced" toml:"https_enforced" yaml:"https_enforced" ini:"https_enforced"`

	// WebSocketHandshakeTimeout is the maximum amount of time the server
	// waits for the WebSocket handshake to complete.
	//
	// The default value is 0.
	WebSocketHandshakeTimeout time.Duration `json:"websocket_handshake_timeout" xml:"websocket_handshake_timeout" toml:"websocket_handshake_timeout" yaml:"websocket_handshake_timeout" ini:"websocket_handshake_timeout"`

	// WebSocketSubprotocols is the supported WebSocket subprotocols of the
	// server.
	//
	// The default value is nil.
	WebSocketSubprotocols []string `json:"websocket_subprotocols" xml:"websocket_subprotocols" toml:"websocket_subprotocols" yaml:"websocket_subprotocols" ini:"websocket_subprotocols"`

	// NotFoundHandler is a `Handler` that returns not found error.
	//
	// The default value is the `DefaultNotFoundHandler`.
	NotFoundHandler func(*Request, *Response) error `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// MethodNotAllowedHandler is a `Handler` that returns method not
	// allowed error.
	//
	// The default value is the `DefaultMethodNotAllowedHandler`.
	MethodNotAllowedHandler func(*Request, *Response) error `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// ErrorHandler is the centralized error handler of the server.
	//
	// The default value is the `DefaultErrorHandler`.
	ErrorHandler func(error, *Request, *Response) `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// ErrorLogger is the `log.Logger` that logs errors that occur in the
	// server.
	//
	// If nil, logging is done via the log package's standard logger.
	//
	// The default value is nil.
	ErrorLogger *log.Logger `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// Pregases is the `Gas` chain stack that performs before routing.
	//
	// The default value is nil.
	Pregases []Gas `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// Gases is the `Gas` chain stack that performs after routing.
	//
	// The default value is nil.
	Gases []Gas `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// AutoPushEnabled indicates whether the auto push is enabled.
	//
	// The default value is false.
	AutoPushEnabled bool `json:"auto_push_enabled" xml:"auto_push_enabled" toml:"auto_push_enabled" yaml:"auto_push_enabled" ini:"auto_push_enabled"`

	// MinifierEnabled indicates whether the minifier is enabled.
	//
	// The default value is false.
	MinifierEnabled bool `json:"minifier_enabled" xml:"minifier_enabled" toml:"minifier_enabled" yaml:"minifier_enabled" ini:"minifier_enabled"`

	// MinifierMIMETypes is the MIME types that will be minified.
	// Unsupported MIME types will be silently ignored.
	//
	// The default value is ["text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "image/svg+xml"].
	MinifierMIMETypes []string `json:"minifier_mime_types" xml:"minifier_mime_types" toml:"minifier_mime_types" yaml:"minifier_mime_types" ini:"minifier_mime_types"`

	// GzipEnabled indicates whether the gzip is enabled.
	//
	// The default value is false.
	GzipEnabled bool `json:"gzip_enabled" xml:"gzip_enabled" toml:"gzip_enabled" yaml:"gzip_enabled" ini:"gzip_enabled"`

	// GzipCompressionLevel is the compression level of the gzip.
	//
	// The default value is `gzip.DefaultCompression`.
	GzipCompressionLevel int `json:"gzip_compression_level" xml:"gzip_compression_level" toml:"gzip_compression_level" yaml:"gzip_compression_level" ini:"gzip_compression_level"`

	// GzipMIMETypes is the MIME types that will be gzipped.
	//
	// The default value is ["text/plain", "text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "image/svg+xml"].
	GzipMIMETypes []string `json:"gzip_mime_types" xml:"gzip_mime_types" toml:"gzip_mime_types" yaml:"gzip_mime_types" ini:"gzip_mime_types"`

	// TemplateRoot is the root of the HTML templates. All the HTML
	// templates inside it will be recursively parsed into the renderer.
	//
	// The default value is "templates".
	TemplateRoot string `json:"template_root" xml:"template_root" toml:"template_root" yaml:"template_root" ini:"template_root"`

	// TemplateExts is the filename extensions of the HTML templates used to
	// distinguish the HTML template files in the `TemplateRoot` when
	// parsing them into the renderer.
	//
	// The default value is [".html"].
	TemplateExts []string `json:"template_exts" xml:"template_exts" toml:"template_exts" yaml:"template_exts" ini:"template_exts"`

	// TemplateLeftDelim is the left side of the HTML template delimiter the
	// renderer renders the HTML templates.
	//
	// The default value is "{{".
	TemplateLeftDelim string `json:"template_left_delim" xml:"template_left_delim" toml:"template_left_delim" yaml:"template_left_delim" ini:"template_left_delim"`

	// TemplateRightDelim is the right side of the HTML template delimiter
	// the renderer renders the HTML templates.
	//
	// The default value is "}}".
	TemplateRightDelim string `json:"template_right_delim" xml:"template_right_delim" toml:"template_right_delim" yaml:"template_right_delim" ini:"template_right_delim"`

	// TemplateFuncMap is the HTML template function map the renderer
	// renders the HTML templates.
	//
	// The default value contains strlen, substr and timefmt.
	TemplateFuncMap template.FuncMap `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

	// CofferEnabled indicates whether the coffer is enabled.
	//
	// The default value is false.
	CofferEnabled bool `json:"coffer_enabled" xml:"coffer_enabled" toml:"coffer_enabled" yaml:"coffer_enabled" ini:"coffer_enabled"`

	// CofferMaxMemoryBytes is the maximum number of bytes of the runtime
	// memory the coffer will use.
	//
	// The default value is 33554432.
	CofferMaxMemoryBytes int `json:"coffer_max_memory_bytes" xml:"coffer_max_memory_bytes" toml:"coffer_max_memory_bytes" yaml:"coffer_max_memory_bytes" ini:"coffer_max_memory_bytes"`

	// AssetRoot is the root of the asset files. All the asset files inside
	// it will be recursively parsed into the coffer.
	//
	// The default value is "assets".
	AssetRoot string `json:"asset_root" xml:"asset_root" toml:"asset_root" yaml:"asset_root" ini:"asset_root"`

	// AssetExts is the filename extensions of the asset files used to
	// distinguish the asset files in the `AssetRoot` when loading them into
	// the coffer.
	//
	// The default value is [".html", ".css", ".js", ".json", ".xml",
	// ".svg", ".jpg", ".jpeg", ".png", ".gif"].
	AssetExts []string `json:"asset_exts" xml:"asset_exts" toml:"asset_exts" yaml:"asset_exts" ini:"asset_exts"`

	// I18nEnabled indicates whether the i18n is enabled.
	//
	// The default value is false.
	I18nEnabled bool `json:"i18n_enabled" xml:"i18n_enabled" toml:"i18n_enabled" yaml:"i18n_enabled" ini:"i18n_enabled"`

	// LocaleRoot is the root of the locale files. All the locale files
	// inside it will be parsed into the i18n.
	//
	// The default value is "locales".
	LocaleRoot string `json:"locale_root" xml:"locale_root" toml:"locale_root" yaml:"locale_root" ini:"locale_root"`

	// LocaleBase is the base of the locale files. It will be used when a
	// locale file cannot be found.
	//
	// The default value is "en-US".
	LocaleBase string `json:"locale_base" xml:"locale_base" toml:"locale_base" yaml:"locale_base" ini:"locale_base"`

	// ConfigFile is the path to the configuration file that will be parsed
	// into the matching fields before starting the server.
	//
	// ".json" -> The configuration file is JSON-based.
	//
	// ".xml" -> The configuration file is XML-based.
	//
	// ".toml" -> The configuration file is TOML-based.
	//
	// ".yaml|.yml" -> The configuration file is YAML-based.
	//
	// ".ini|.cfg|.conf|.txt" -> The configuration file is INI-based.
	//
	// The default value is "".
	ConfigFile string `json:"-" xml:"-" toml:"-" yaml:"-" ini:"-"`

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
		b, err := ioutil.ReadFile(a.ConfigFile)
		if err != nil {
			return err
		}

		switch strings.ToLower(filepath.Ext(a.ConfigFile)) {
		case ".json":
			err = json.Unmarshal(b, a)
		case ".xml":
			err = xml.Unmarshal(b, a)
		case ".toml":
			err = toml.Unmarshal(b, a)
		case ".yaml", ".yml":
			err = yaml.Unmarshal(b, a)
		case ".ini", ".cfg", ".conf", ".txt":
			var cfg *ini.File
			if cfg, err = ini.Load(b); err != nil {
				return err
			}

			err = cfg.MapTo(a)
		}

		if err != nil {
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

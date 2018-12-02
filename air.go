package air

import (
	"compress/gzip"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/net/http2"
)

// Air is the top-level struct of this framework.
type Air struct {
	// AppName is the name of the current web application.
	//
	// The default value is "air".
	//
	// It is called "app_name" when it is used as a configuration item.
	AppName string

	// MaintainerEmail is the e-mail address of the one who is responsible
	// for maintaining the current web application.
	//
	// The default value is "".
	//
	// It is called "maintainer_email" when it is used as a configuration
	// item.
	MaintainerEmail string

	// DebugMode indicates whether the current web application is in debug
	// mode.
	//
	// ATTENTION: Some features will be affected in debug mode.
	//
	// The default value is false.
	//
	// It is called "debug_mode" when it is used as a configuration item.
	DebugMode bool

	// LoggerLowestLevel is the lowest level of the logger.
	//
	// It only works when the `DebugMode` is false.
	//
	// The default value is the `LoggerLevelDebug`.
	//
	// It is called "logger_lowest_level" when it is used as a configuration
	// item.
	LoggerLowestLevel LoggerLevel

	// LoggerOutput is the output destination of the logger.
	//
	// The default value is the `os.Stdout`.
	LoggerOutput io.Writer

	// Address is the TCP address that the server listens on.
	//
	// The default value is "localhost:2333".
	//
	// It is called "address" when it is used as a configuration item.
	Address string

	// HostWhitelist is the hosts allowed by the server.
	//
	// It only works when the `DebugMode` is false.
	//
	// The default value is nil.
	//
	// It is called "host_whitelist" when it is used as a configuration
	// item.
	HostWhitelist []string

	// ReadTimeout is the maximum duration the server reads the request.
	//
	// The default value is 0.
	//
	// It is called "read_timeout" when it is used as a configuration item.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed the server reads the
	// request headers.
	//
	// The default value is 0.
	//
	// It is called "read_header_timeout" when it is used as a configuration
	// item.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration the server writes an response.
	//
	// The default value is 0.
	//
	// It is called "write_timeout" when it is used as a configuration item.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time the server waits for the
	// next request. If it is zero, the value of `ReadTimeout` is used. If
	// both are zero, the value `ReadHeaderTimeout` is used.
	//
	// The default value is 0.
	//
	// It is called "idle_timeout" when it is used as a configuration item.
	IdleTimeout time.Duration

	// MaxHeaderBytes is the maximum number of bytes the server will read
	// parsing the request header's names and values, including the request
	// line.
	//
	// The default value is 1048576.
	//
	// It is called "max_header_bytes" when it is used as a configuration
	// item.
	MaxHeaderBytes int

	// TLSCertFile is the path to the TLS certificate file used when
	// starting the server.
	//
	// The default value is "".
	//
	// It is called "tls_cert_file" when it is used as a configuration item.
	TLSCertFile string

	// TLSKeyFile is the path to the TLS key file used when starting the
	// server.
	//
	// The default value is "".
	//
	// It is called "tls_key_file" when it is used as a configuration item.
	TLSKeyFile string

	// ACMEEnabled indicates whether the ACME is enabled.
	//
	// It only works when the `DebugMode` is false and both the
	// `TLSCertFile` and the `TLSKeyFile` are empty.
	//
	// The default value is false.
	//
	// It is called "acme_enabled" when it is used as a configuration item.
	ACMEEnabled bool

	// ACMECertRoot is the root of the ACME certificates.
	//
	// The default value is "acme-certs".
	//
	// It is called "acme_cert_root" when it is used as a configuration
	// item.
	ACMECertRoot string

	// HTTPSEnforced indicates whether the HTTPS is enforced.
	//
	// The default value is false.
	//
	// It is called "https_enforced" when it is used as a configuration
	// item.
	HTTPSEnforced bool

	// WebSocketHandshakeTimeout is the maximum amount of time the server
	// waits for the WebSocket handshake to complete.
	//
	// The default value is 0.
	//
	// It is called "websocket_handshake_timeout" when it is used as a
	// configuration item.
	WebSocketHandshakeTimeout time.Duration

	// WebSocketSubprotocols is the server's supported WebSocket
	// subprotocols.
	//
	// The default value is nil.
	//
	// It is called "websocket_subprotocols" when it is used as a
	// configuration item.
	WebSocketSubprotocols []string

	// NotFoundHandler is a `Handler` that returns not found error.
	//
	// The default value is the `DefaultNotFoundHandler`.
	NotFoundHandler func(*Request, *Response) error

	// MethodNotAllowedHandler is a `Handler` that returns method not
	// allowed error.
	//
	// The default value is the `DefaultMethodNotAllowedHandler`.
	MethodNotAllowedHandler func(*Request, *Response) error

	// ErrorHandler is the centralized error handler for the server.
	//
	// The default value is the `DefaultErrorHandler`.
	ErrorHandler func(error, *Request, *Response)

	// Pregases is the `Gas` chain that performs before routing.
	//
	// The default value is nil.
	Pregases []Gas

	// Gases is the `Gas` chain that performs after routing.
	//
	// The default value is nil.
	Gases []Gas

	// AutoPushEnabled indicates whether the auto push is enabled.
	//
	// The default value is false.
	//
	// It is called "auto_push_enabled" when it is used as a configuration
	// item.
	AutoPushEnabled bool

	// MinifierEnabled indicates whether the minifier is enabled.
	//
	// The default value is false.
	//
	// It is called "minifier_enabled" when it is used as a configuration
	// item.
	MinifierEnabled bool

	// MinifierMIMETypes is the MIME types that will be minified.
	// Unsupported MIME types will be silently ignored.
	//
	// The default value is ["text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "image/svg+xml"].
	//
	// It is called "minifier_mime_types" when it is used as a configuration
	// item.
	MinifierMIMETypes []string

	// GzipEnabled indicates whether the gzip is enabled.
	//
	// The default value is false.
	//
	// It is called "gzip_enabled" when it is used as a configuration item.
	GzipEnabled bool

	// GzipCompressionLevel is the compression level of the gzip.
	//
	// The default value is `gzip.DefaultCompression`.
	//
	// It is called "gzip_compression_level" when it is used as a
	// configuration item.
	GzipCompressionLevel int

	// GzipMIMETypes is the MIME types that will be gzipped.
	//
	// The default value is ["text/plain", "text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "image/svg+xml"].
	//
	// It is called "gzip_mime_types" when it is used as a configuration
	// item.
	GzipMIMETypes []string

	// TemplateRoot is the root of the HTML templates. All the HTML
	// templates inside it will be recursively parsed into the renderer.
	//
	// The default value is "templates".
	//
	// It is called "template_root" when it is used as a configuration item.
	TemplateRoot string

	// TemplateExts is the filename extensions of the HTML templates used to
	// distinguish the HTML template files in the `TemplateRoot` when
	// parsing them into the renderer.
	//
	// The default value is [".html"].
	//
	// It is called "template_exts" when it is used as a configuration item.
	TemplateExts []string

	// TemplateLeftDelim is the left side of the HTML template delimiter the
	// renderer renders the HTML templates.
	//
	// The default value is "{{".
	//
	// It is called "template_left_delim" when it is used as a configuration
	// item.
	TemplateLeftDelim string

	// TemplateRightDelim is the right side of the HTML template delimiter
	// the renderer renders the HTML templates.
	//
	// The default value is "}}".
	//
	// It is called "template_right_delim" when it is used as a
	// configuration item.
	TemplateRightDelim string

	// TemplateFuncMap is the HTML template function map the renderer
	// renders the HTML templates.
	//
	// The default value contains strlen, substr and timefmt.
	TemplateFuncMap map[string]interface{}

	// CofferEnabled indicates whether the coffer is enabled.
	//
	// The default value is false.
	//
	// It is called "coffer_enabled" when it is used as a configuration
	// item.
	CofferEnabled bool

	// CofferMaxMemoryBytes is the maximum number of bytes of the runtime
	// memory the coffer will use.
	//
	// The default value is 33554432.
	//
	// It is called "coffer_max_memory_bytes" when it is used as a
	// configuration item.
	CofferMaxMemoryBytes int

	// AssetRoot is the root of the asset files. All the asset files inside
	// it will be recursively parsed into the coffer.
	//
	// The default value is "assets".
	//
	// It is called "asset_root" when it is used as a configuration item.
	AssetRoot string

	// AssetExts is the filename extensions of the asset files used to
	// distinguish the asset files in the `AssetRoot` when loading them into
	// the coffer.
	//
	// The default value is [".html", ".css", ".js", ".json", ".xml",
	// ".svg", ".jpg", ".jpeg", ".png", ".gif"].
	//
	// It is called "asset_exts" when it is used as a configuration item.
	AssetExts []string

	// I18nEnabled indicates whether the i18n is enabled.
	//
	// The default value is false.
	//
	// It is called "i18n_enabled" when it is used as a configuration item.
	I18nEnabled bool

	// LocaleRoot is the root of the locale files. All the locale files
	// inside it will be parsed into the i18n.
	//
	// The default value is "locales".
	//
	// It is called "locale_root" when it is used as a configuration item.
	LocaleRoot string

	// LocaleBase is the base of the locale files. It will be used when a
	// locale file cannot be found.
	//
	// The default value is "en-US".
	//
	// It is called "locale_base" when it is used as a configuration item.
	LocaleBase string

	// ConfigFile is the TOML-based configuration file that will be parsed
	// into the matching configuration items before starting the server.
	//
	// The default value is "".
	ConfigFile string

	logger                 *logger
	errorLogWriter         *errorLogWriter
	server                 *server
	router                 *router
	binder                 *binder
	minifier               *minifier
	renderer               *renderer
	coffer                 *coffer
	i18n                   *i18n
	reverseProxyTransport  *http.Transport
	reverseProxyBufferPool *reverseProxyBufferPool
}

// Default is the default instance of the `Air`.
var Default = New()

// New returns a new instance of the `Air`.
func New() *Air {
	a := &Air{
		AppName:                 "air",
		LoggerOutput:            os.Stdout,
		Address:                 "localhost:2333",
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
		TemplateFuncMap: map[string]interface{}{
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

	a.logger = newLogger(a)
	a.errorLogWriter = newErrorLogWriter(a)
	a.server = newServer(a)
	a.router = newRouter(a)
	a.binder = newBinder(a)
	a.minifier = newMinifier(a)
	a.renderer = newRenderer(a)
	a.coffer = newCoffer(a)
	a.i18n = newI18n(a)
	a.reverseProxyTransport = newReverseProxyTransport()
	a.reverseProxyBufferPool = newReverseProxyBufferPool()

	return a
}

// DEBUG logs the msg at the `LoggerLevelDebug` with the optional extras.
func (a *Air) DEBUG(msg string, extras ...map[string]interface{}) {
	a.logger.log(LoggerLevelDebug, msg, extras...)
}

// INFO logs the msg at the `LoggerLevelInfo` with the optional extras.
func (a *Air) INFO(msg string, extras ...map[string]interface{}) {
	a.logger.log(LoggerLevelInfo, msg, extras...)
}

// WARN logs the msg at the `LoggerLevelWarn` with the optional extras.
func (a *Air) WARN(msg string, extras ...map[string]interface{}) {
	a.logger.log(LoggerLevelWarn, msg, extras...)
}

// ERROR logs the msg at the `LoggerLevelError` with the optional extras.
func (a *Air) ERROR(msg string, extras ...map[string]interface{}) {
	a.logger.log(LoggerLevelError, msg, extras...)
}

// FATAL logs the msg at the `LoggerLevelFatal` with the optional extras
// followed by a call to `os.Exit(1)`.
func (a *Air) FATAL(msg string, extras ...map[string]interface{}) {
	a.logger.log(LoggerLevelFatal, msg, extras...)
	os.Exit(1)
}

// PANIC logs the msg at the `LoggerLevelPanic` with the optional extras
// followed by a call to `panic()`.
func (a *Air) PANIC(msg string, extras ...map[string]interface{}) {
	a.logger.log(LoggerLevelPanic, msg, extras...)
	panic(msg)
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

// STATIC registers a new route with the path prefix to serve the static files
// from the root with the optional route-level gases.
func (a *Air) STATIC(prefix, root string, gases ...Gas) {
	if hasLastSlash(prefix) {
		prefix += "*"
	} else {
		prefix += "/*"
	}

	if root == "" {
		root = "."
	}

	h := func(req *Request, res *Response) error {
		err := res.WriteFile(filepath.Join(
			root,
			req.Param("*").Value().String(),
		))
		if os.IsNotExist(err) {
			return a.NotFoundHandler(req, res)
		}

		return err
	}

	a.BATCH([]string{http.MethodGet, http.MethodHead}, prefix, h, gases...)
}

// FILE registers a new route with the path to serve a static file with the
// filename and the optional route-level gases.
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

// Group returns a new instance of the `Group` with the prefix and the optional
// group-level gases.
func (a *Air) Group(prefix string, gases ...Gas) *Group {
	return &Group{
		Air:    a,
		Prefix: prefix,
		Gases:  gases,
	}
}

// Serve starts the server.
func (a *Air) Serve() error {
	if a.ConfigFile == "" {
		return a.server.serve()
	}

	m := map[string]toml.Primitive{}
	md, err := toml.DecodeFile(a.ConfigFile, &m)
	if err != nil {
		return err
	}

	if p, ok := m["app_name"]; ok {
		if err := md.PrimitiveDecode(p, &a.AppName); err != nil {
			return err
		}
	}

	if p, ok := m["maintainer_email"]; ok {
		err := md.PrimitiveDecode(p, &a.MaintainerEmail)
		if err != nil {
			return err
		}
	}

	if p, ok := m["debug_mode"]; ok {
		if err := md.PrimitiveDecode(p, &a.DebugMode); err != nil {
			return err
		}
	}

	if p, ok := m["logger_lowest_level"]; ok {
		lll := ""
		if err := md.PrimitiveDecode(p, &lll); err != nil {
			return err
		}

		switch lll {
		case LoggerLevelDebug.String():
			a.LoggerLowestLevel = LoggerLevelDebug
		case LoggerLevelInfo.String():
			a.LoggerLowestLevel = LoggerLevelInfo
		case LoggerLevelWarn.String():
			a.LoggerLowestLevel = LoggerLevelWarn
		case LoggerLevelError.String():
			a.LoggerLowestLevel = LoggerLevelError
		case LoggerLevelFatal.String():
			a.LoggerLowestLevel = LoggerLevelFatal
		case LoggerLevelPanic.String():
			a.LoggerLowestLevel = LoggerLevelPanic
		case LoggerLevelOff.String():
			a.LoggerLowestLevel = LoggerLevelOff
		}
	}

	if p, ok := m["address"]; ok {
		if err := md.PrimitiveDecode(p, &a.Address); err != nil {
			return err
		}
	}

	if p, ok := m["host_whitelist"]; ok {
		a.HostWhitelist = a.HostWhitelist[:0]
		if err := md.PrimitiveDecode(p, &a.HostWhitelist); err != nil {
			return err
		}
	}

	if p, ok := m["read_timeout"]; ok {
		if err := md.PrimitiveDecode(p, &a.ReadTimeout); err != nil {
			return err
		}
	}

	if p, ok := m["read_header_timeout"]; ok {
		err := md.PrimitiveDecode(p, &a.ReadHeaderTimeout)
		if err != nil {
			return err
		}
	}

	if p, ok := m["write_timeout"]; ok {
		if err := md.PrimitiveDecode(p, &a.WriteTimeout); err != nil {
			return err
		}
	}

	if p, ok := m["idle_timeout"]; ok {
		if err := md.PrimitiveDecode(p, &a.IdleTimeout); err != nil {
			return err
		}
	}

	if p, ok := m["max_header_bytes"]; ok {
		if err := md.PrimitiveDecode(p, &a.MaxHeaderBytes); err != nil {
			return err
		}
	}

	if p, ok := m["tls_cert_file"]; ok {
		if err := md.PrimitiveDecode(p, &a.TLSCertFile); err != nil {
			return err
		}
	}

	if p, ok := m["tls_key_file"]; ok {
		if err := md.PrimitiveDecode(p, &a.TLSKeyFile); err != nil {
			return err
		}
	}

	if p, ok := m["acme_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &a.ACMEEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["acme_cert_root"]; ok {
		if err := md.PrimitiveDecode(p, &a.ACMECertRoot); err != nil {
			return err
		}
	}

	if p, ok := m["https_enforced"]; ok {
		if err := md.PrimitiveDecode(p, &a.HTTPSEnforced); err != nil {
			return err
		}
	}

	if p, ok := m["websocket_handshake_timeout"]; ok {
		err := md.PrimitiveDecode(p, &a.WebSocketHandshakeTimeout)
		if err != nil {
			return err
		}
	}

	if p, ok := m["websocket_subprotocols"]; ok {
		a.WebSocketSubprotocols = a.WebSocketSubprotocols[:0]
		err := md.PrimitiveDecode(p, &a.WebSocketSubprotocols)
		if err != nil {
			return err
		}
	}

	if p, ok := m["auto_push_enabled"]; ok {
		err := md.PrimitiveDecode(p, &a.AutoPushEnabled)
		if err != nil {
			return err
		}
	}

	if p, ok := m["minifier_enabled"]; ok {
		err := md.PrimitiveDecode(p, &a.MinifierEnabled)
		if err != nil {
			return err
		}
	}

	if p, ok := m["minifier_mime_types"]; ok {
		a.MinifierMIMETypes = a.MinifierMIMETypes[:0]
		err := md.PrimitiveDecode(p, &a.MinifierMIMETypes)
		if err != nil {
			return err
		}
	}

	if p, ok := m["gzip_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &a.GzipEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["gzip_compression_level"]; ok {
		err := md.PrimitiveDecode(p, &a.GzipCompressionLevel)
		if err != nil {
			return err
		}
	}

	if p, ok := m["gzip_mime_types"]; ok {
		a.GzipMIMETypes = a.GzipMIMETypes[:0]
		if err := md.PrimitiveDecode(p, &a.GzipMIMETypes); err != nil {
			return err
		}
	}

	if p, ok := m["template_root"]; ok {
		if err := md.PrimitiveDecode(p, &a.TemplateRoot); err != nil {
			return err
		}
	}

	if p, ok := m["template_exts"]; ok {
		a.TemplateExts = a.TemplateExts[:0]
		if err := md.PrimitiveDecode(p, &a.TemplateExts); err != nil {
			return err
		}
	}

	if p, ok := m["template_left_delim"]; ok {
		err := md.PrimitiveDecode(p, &a.TemplateLeftDelim)
		if err != nil {
			return err
		}
	}

	if p, ok := m["template_right_delim"]; ok {
		err := md.PrimitiveDecode(p, &a.TemplateRightDelim)
		if err != nil {
			return err
		}
	}

	if p, ok := m["coffer_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &a.CofferEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["coffer_max_memory_bytes"]; ok {
		err := md.PrimitiveDecode(p, &a.CofferMaxMemoryBytes)
		if err != nil {
			return err
		}
	}

	if p, ok := m["asset_root"]; ok {
		if err := md.PrimitiveDecode(p, &a.AssetRoot); err != nil {
			return err
		}
	}

	if p, ok := m["asset_exts"]; ok {
		a.AssetExts = a.AssetExts[:0]
		if err := md.PrimitiveDecode(p, &a.AssetExts); err != nil {
			return err
		}
	}

	if p, ok := m["i18n_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &a.I18nEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["locale_root"]; ok {
		if err := md.PrimitiveDecode(p, &a.LocaleRoot); err != nil {
			return err
		}
	}

	if p, ok := m["locale_base"]; ok {
		if err := md.PrimitiveDecode(p, &a.LocaleBase); err != nil {
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
	if !res.Written {
		if res.Status < http.StatusBadRequest {
			res.Status = http.StatusInternalServerError
		}

		if req.Method == http.MethodGet ||
			req.Method == http.MethodHead {
			res.Header.Del("ETag")
			res.Header.Del("Last-Modified")
		}
	}

	if res.ContentLength == 0 {
		m := err.Error()
		if !req.Air.DebugMode &&
			res.Status == http.StatusInternalServerError {
			m = http.StatusText(res.Status)
		}

		res.WriteString(m)
	}
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
	elw.a.ERROR(strings.TrimSuffix(string(b), "\n"))
	return len(b), nil
}

// newReverseProxyTransport returns a new instance of the `http.Transport` with
// reverse proxy support.
func newReverseProxyTransport() *http.Transport {
	rpt := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConnsPerHost:   200,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	rpt.RegisterProtocol("ws", newWSTransport(false))
	rpt.RegisterProtocol("wss", newWSTransport(true))
	rpt.RegisterProtocol("grpc", newGRPCTransport(false))
	rpt.RegisterProtocol("grpcs", newGRPCTransport(true))

	return rpt
}

// wsTransport is a transport with WebSocket support.
type wsTransport struct {
	*http.Transport

	tlsed bool
}

// newWSTransport returns a new instance of the `wsTransport` with the tlsed.
func newWSTransport(tlsed bool) *wsTransport {
	return &wsTransport{
		Transport: &http.Transport{},

		tlsed: tlsed,
	}
}

// RoundTrip implements the `http.Transport`.
func (wst *wsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if wst.tlsed {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}

	return wst.Transport.RoundTrip(req)
}

// grpcTransport is a transport with gRPC support.
type grpcTransport struct {
	*http2.Transport

	tlsed bool
}

// newGRPCTransport returns a new instance of the `grpcTransport` with the
// tlsed.
func newGRPCTransport(tlsed bool) *grpcTransport {
	gt := &grpcTransport{
		Transport: &http2.Transport{},

		tlsed: tlsed,
	}
	if !tlsed {
		gt.DialTLS = func(
			network string,
			address string,
			_ *tls.Config,
		) (net.Conn, error) {
			return net.Dial(network, address)
		}

		gt.AllowHTTP = true
	}

	return gt
}

// RoundTrip implements the `http2.Transport`.
func (gt *grpcTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if gt.tlsed {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}

	return gt.Transport.RoundTrip(req)
}

// reverseProxyBufferPool is a buffer pool for the reverse proxy.
type reverseProxyBufferPool struct {
	pool sync.Pool
}

// newReverseProxyBufferPool returns a new instance of the
// `reverseProxyBufferPool`.
func newReverseProxyBufferPool() *reverseProxyBufferPool {
	return &reverseProxyBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 32<<20)
			},
		},
	}
}

// Get implements the `httputil.BufferPool`.
func (rpbp *reverseProxyBufferPool) Get() []byte {
	return rpbp.pool.Get().([]byte)
}

// Put implements the `httputil.BufferPool`.
func (rpbp *reverseProxyBufferPool) Put(bytes []byte) {
	rpbp.pool.Put(bytes)
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

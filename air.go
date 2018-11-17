package air

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// AppName is the name of the current web application.
//
// It is called "app_name" when it is used as a configuration item.
var AppName = "air"

// MaintainerEmail is the e-mail address of the one who is responsible for
// maintaining the current web application.
//
// It is called "maintainer_email" when it is used as a configuration item.
var MaintainerEmail = ""

// DebugMode indicates whether the current web application is in debug mode.
//
// ATTENTION: Some features will be affected in debug mode.
//
// It is called "debug_mode" when it is used as a configuration item.
var DebugMode = false

// LoggerLowestLevel is the lowest level of the logger.
//
// It only works when the `DebugMode` is false.
//
// It is called "logger_lowest_level" when it is used as a configuration item.
var LoggerLowestLevel = LoggerLevelDebug

// LoggerOutput is the output destination of the logger.
var LoggerOutput = io.Writer(os.Stdout)

// Address is the TCP address that the server listens on.
//
// It is called "address" when it is used as a configuration item.
var Address = "localhost:2333"

// HostWhitelist is the hosts allowed by the server.
//
// It only works when the `DebugMode` is false.
//
// It is called "host_whitelist" when it is used as a configuration item.
var HostWhitelist = []string{}

// ReadTimeout is the maximum duration the server reads the request.
//
// It is called "read_timeout" when it is used as a configuration item.
var ReadTimeout = time.Duration(0)

// ReadHeaderTimeout is the amount of time allowed the server reads the request
// headers.
//
// It is called "read_header_timeout" when it is used as a configuration item.
var ReadHeaderTimeout = time.Duration(0)

// WriteTimeout is the maximum duration the server writes an response.
//
// It is called "write_timeout" when it is used as a configuration item.
var WriteTimeout = time.Duration(0)

// IdleTimeout is the maximum amount of time the server waits for the next
// request. If it is zero, the value of `ReadTimeout` is used. If both are zero,
// `ReadHeaderTimeout` is used.
//
// It is called "idle_timeout" when it is used as a configuration item.
var IdleTimeout = time.Duration(0)

// MaxHeaderBytes is the maximum number of bytes the server will read parsing
// the request header's names and values, including the request line.
//
// It is called "max_header_bytes" when it is used as a configuration item.
var MaxHeaderBytes = 1 << 20

// TLSCertFile is the path to the TLS certificate file used when starting the
// server.
//
// It is called "tls_cert_file" when it is used as a configuration item.
var TLSCertFile = ""

// TLSKeyFile is the path to the TLS key file used when starting the server.
//
// It is called "tls_key_file" when it is used as a configuration item.
var TLSKeyFile = ""

// ACMEEnabled indicates whether the ACME is enabled.
//
// It only works when the `DebugMode` is false and both the `TLSCertFile` and
// the `TLSKeyFile` are empty.
//
// It is called "acme_enabled" when it is used as a configuration item.
var ACMEEnabled = false

// ACMECertRoot is the root of the ACME certificates.
//
// It is called "acme_cert_root" when it is used as a configuration item.
var ACMECertRoot = "acme-certs"

// HTTPSEnforced indicates whether the HTTPS is enforced.
//
// It is called "https_enforced" when it is used as a configuration item.
var HTTPSEnforced = false

// WebSocketHandshakeTimeout is the maximum amount of time the server waits for
// the WebSocket handshake to complete.
//
// It is called "websocket_handshake_timeout" when it is used as a configuration
// item.
var WebSocketHandshakeTimeout = time.Duration(0)

// WebSocketSubprotocols is the server's supported WebSocket subprotocols.
//
// It is called "websocket_subprotocols" when it is used as a configuration
// item.
var WebSocketSubprotocols = []string{}

// ErrorHandler is the centralized error handler for the server.
var ErrorHandler = func(err error, req *Request, res *Response) {
	if res.Written {
		return
	}

	if res.Status < http.StatusBadRequest {
		res.Status = http.StatusInternalServerError
	}

	m := err.Error()
	if !DebugMode && res.Status == http.StatusInternalServerError {
		m = http.StatusText(res.Status)
	}

	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		res.Header.Del("ETag")
		res.Header.Del("Last-Modified")
	}

	res.WriteString(m)
}

// Pregases is the `Gas` chain that performs before routing.
var Pregases = []Gas{}

// Gases is the `Gas` chain that performs after routing.
var Gases = []Gas{}

// AutoPushEnabled indicates whether the auto push is enabled.
//
// It is called "auto_push_enabled" when it is used as a configuration item.
var AutoPushEnabled = false

// MinifierEnabled indicates whether the minifier is enabled.
//
// It is called "minifier_enabled" when it is used as a configuration item.
var MinifierEnabled = false

// TemplateRoot is the root of the HTML templates. All the HTML templates inside
// it will be recursively parsed into the renderer.
//
// It is called "template_root" when it is used as a configuration item.
var TemplateRoot = "templates"

// TemplateExts is the filename extensions of the HTML templates used to
// distinguish the HTML template files in the `TemplateRoot` when parsing them
// into the renderer.
//
// It is called "template_exts" when it is used as a configuration item.
var TemplateExts = []string{".html"}

// TemplateLeftDelim is the left side of the HTML template delimiter the
// renderer renders the HTML templates.
//
// It is called "template_left_delim" when it is used as a configuration item.
var TemplateLeftDelim = "{{"

// TemplateRightDelim is the right side of the HTML template delimiter the
// renderer renders the HTML templates.
//
// It is called "template_right_delim" when it is used as a configuration item.
var TemplateRightDelim = "}}"

// TemplateFuncMap is the HTML template function map the renderer renders the
// HTML templates.
var TemplateFuncMap = map[string]interface{}{
	"strlen":  strlen,
	"substr":  substr,
	"timefmt": timefmt,
}

// CofferEnabled indicates whether the coffer is enabled.
//
// It is called "coffer_enabled" when it is used as a configuration item.
var CofferEnabled = false

// CofferMaxMemoryBytes is the maximum number of bytes of the runtime memory
// the coffer will use.
//
// It is called "coffer_max_memory_bytes" when it is used as a configuration
// item.
var CofferMaxMemoryBytes = 32 << 20

// AssetRoot is the root of the asset files. All the asset files inside it will
// be recursively parsed into the coffer.
//
// It is called "asset_root" when it is used as a configuration item.
var AssetRoot = "assets"

// AssetCacheRoot is the root of the asset cache files.
//
// It is called "asset_cache_root" when it is used as a configuration item.
var AssetCacheRoot = AssetRoot + "/.cache"

// AssetExts is the filename extensions of the asset files used to distinguish
// the asset files in the `AssetRoot` when loading them into the coffer.
//
// It is called "asset_exts" when it is used as a configuration item.
var AssetExts = []string{
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
}

// I18nEnabled indicates whether the i18n is enabled.
//
// It is called "i18n_enabled" when it is used as a configuration item.
var I18nEnabled = false

// LocaleRoot is the root of the locale files. All the locale files inside it
// will be parsed into the i18n.
//
// It is called "locale_root" when it is used as a configuration item.
var LocaleRoot = "locales"

// LocaleBase is the base of the locale files. It will be used when a locale
// file cannot be found.
//
// It is called "locale_base" when it is used as a configuration item.
var LocaleBase = "en-US"

// ConfigFile is the TOML-based configuration file that will be parsed into the
// matching configuration items before starting the server.
var ConfigFile = ""

// DEBUG logs the msg at the `LoggerLevelDebug` with the optional extras.
func DEBUG(msg string, extras ...map[string]interface{}) {
	theLogger.log(LoggerLevelDebug, msg, extras...)
}

// INFO logs the msg at the `LoggerLevelInfo` with the optional extras.
func INFO(msg string, extras ...map[string]interface{}) {
	theLogger.log(LoggerLevelInfo, msg, extras...)
}

// WARN logs the msg at the `LoggerLevelWarn` with the optional extras.
func WARN(msg string, extras ...map[string]interface{}) {
	theLogger.log(LoggerLevelWarn, msg, extras...)
}

// ERROR logs the msg at the `LoggerLevelError` with the optional extras.
func ERROR(msg string, extras ...map[string]interface{}) {
	theLogger.log(LoggerLevelError, msg, extras...)
}

// FATAL logs the msg at the `LoggerLevelFatal` with the optional extras
// followed by a call to `os.Exit(1)`.
func FATAL(msg string, extras ...map[string]interface{}) {
	theLogger.log(LoggerLevelFatal, msg, extras...)
	os.Exit(1)
}

// PANIC logs the msg at the `LoggerLevelPanic` with the optional extras
// followed by a call to `panic()`.
func PANIC(msg string, extras ...map[string]interface{}) {
	theLogger.log(LoggerLevelPanic, msg, extras...)
	panic(msg)
}

// GET registers a new GET route for the path with the matching h in the router
// with the optional route-level gases.
func GET(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodGet, path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router with the optional route-level gases.
func HEAD(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodHead, path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router with the optional route-level gases.
func POST(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodPost, path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// with the optional route-level gases.
func PUT(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodPut, path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router with the optional route-level gases.
func PATCH(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodPatch, path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router with the optional route-level gases.
func DELETE(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodDelete, path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router with the optional route-level gases.
func CONNECT(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodConnect, path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router with the optional route-level gases.
func OPTIONS(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodOptions, path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router with the optional route-level gases.
func TRACE(path string, h Handler, gases ...Gas) {
	theRouter.register(http.MethodTrace, path, h, gases...)
}

// STATIC registers a new route with the path prefix to serve the static files
// from the root with the optional route-level gases.
func STATIC(prefix, root string, gases ...Gas) {
	h := func(req *Request, res *Response) error {
		err := res.WriteFile(filepath.Join(
			root,
			req.Param("*").Value().String(),
		))
		if os.IsNotExist(err) {
			return NotFoundHandler(req, res)
		}

		return err
	}

	GET(prefix+"*", h, gases...)
	HEAD(prefix+"*", h, gases...)
}

// FILE registers a new route with the path to serve a static file with the
// filename and the optional route-level gases.
func FILE(path, filename string, gases ...Gas) {
	h := func(req *Request, res *Response) error {
		err := res.WriteFile(filename)
		if os.IsNotExist(err) {
			return NotFoundHandler(req, res)
		}

		return err
	}

	GET(path, h, gases...)
	HEAD(path, h, gases...)
}

// Serve starts the server.
func Serve() error {
	if ConfigFile == "" {
		return theServer.serve()
	}

	m := map[string]toml.Primitive{}
	md, err := toml.DecodeFile(ConfigFile, &m)
	if err != nil {
		return err
	}

	if p, ok := m["app_name"]; ok {
		if err := md.PrimitiveDecode(p, &AppName); err != nil {
			return err
		}
	}

	if p, ok := m["maintainer_email"]; ok {
		if err := md.PrimitiveDecode(p, &MaintainerEmail); err != nil {
			return err
		}
	}

	if p, ok := m["debug_mode"]; ok {
		if err := md.PrimitiveDecode(p, &DebugMode); err != nil {
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
			LoggerLowestLevel = LoggerLevelDebug
		case LoggerLevelInfo.String():
			LoggerLowestLevel = LoggerLevelInfo
		case LoggerLevelWarn.String():
			LoggerLowestLevel = LoggerLevelWarn
		case LoggerLevelError.String():
			LoggerLowestLevel = LoggerLevelError
		case LoggerLevelFatal.String():
			LoggerLowestLevel = LoggerLevelFatal
		case LoggerLevelPanic.String():
			LoggerLowestLevel = LoggerLevelPanic
		case LoggerLevelOff.String():
			LoggerLowestLevel = LoggerLevelOff
		}
	}

	if p, ok := m["address"]; ok {
		if err := md.PrimitiveDecode(p, &Address); err != nil {
			return err
		}
	}

	if p, ok := m["host_whitelist"]; ok {
		HostWhitelist = HostWhitelist[:0]
		if err := md.PrimitiveDecode(p, &HostWhitelist); err != nil {
			return err
		}
	}

	if p, ok := m["read_timeout"]; ok {
		if err := md.PrimitiveDecode(p, &ReadTimeout); err != nil {
			return err
		}
	}

	if p, ok := m["read_header_timeout"]; ok {
		err := md.PrimitiveDecode(p, &ReadHeaderTimeout)
		if err != nil {
			return err
		}
	}

	if p, ok := m["write_timeout"]; ok {
		if err := md.PrimitiveDecode(p, &WriteTimeout); err != nil {
			return err
		}
	}

	if p, ok := m["idle_timeout"]; ok {
		if err := md.PrimitiveDecode(p, &IdleTimeout); err != nil {
			return err
		}
	}

	if p, ok := m["max_header_bytes"]; ok {
		if err := md.PrimitiveDecode(p, &MaxHeaderBytes); err != nil {
			return err
		}
	}

	if p, ok := m["tls_cert_file"]; ok {
		if err := md.PrimitiveDecode(p, &TLSCertFile); err != nil {
			return err
		}
	}

	if p, ok := m["tls_key_file"]; ok {
		if err := md.PrimitiveDecode(p, &TLSKeyFile); err != nil {
			return err
		}
	}

	if p, ok := m["acme_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &ACMEEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["acme_cert_root"]; ok {
		if err := md.PrimitiveDecode(p, &ACMECertRoot); err != nil {
			return err
		}
	}

	if p, ok := m["https_enforced"]; ok {
		if err := md.PrimitiveDecode(p, &HTTPSEnforced); err != nil {
			return err
		}
	}

	if p, ok := m["websocket_handshake_timeout"]; ok {
		err := md.PrimitiveDecode(p, &WebSocketHandshakeTimeout)
		if err != nil {
			return err
		}
	}

	if p, ok := m["websocket_subprotocols"]; ok {
		WebSocketSubprotocols = WebSocketSubprotocols[:0]
		err := md.PrimitiveDecode(p, &WebSocketSubprotocols)
		if err != nil {
			return err
		}
	}

	if p, ok := m["auto_push_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &AutoPushEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["minifier_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &MinifierEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["template_root"]; ok {
		if err := md.PrimitiveDecode(p, &TemplateRoot); err != nil {
			return err
		}
	}

	if p, ok := m["template_exts"]; ok {
		TemplateExts = TemplateExts[:0]
		if err := md.PrimitiveDecode(p, &TemplateExts); err != nil {
			return err
		}
	}

	if p, ok := m["template_left_delim"]; ok {
		err := md.PrimitiveDecode(p, &TemplateLeftDelim)
		if err != nil {
			return err
		}
	}

	if p, ok := m["template_right_delim"]; ok {
		err := md.PrimitiveDecode(p, &TemplateRightDelim)
		if err != nil {
			return err
		}
	}

	if p, ok := m["coffer_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &CofferEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["coffer_max_memory_bytes"]; ok {
		err := md.PrimitiveDecode(p, &CofferMaxMemoryBytes)
		if err != nil {
			return err
		}
	}

	if p, ok := m["asset_root"]; ok {
		if err := md.PrimitiveDecode(p, &AssetRoot); err != nil {
			return err
		}
	}

	if p, ok := m["asset_cache_root"]; ok {
		if err := md.PrimitiveDecode(p, &AssetCacheRoot); err != nil {
			return err
		}
	}

	if p, ok := m["asset_exts"]; ok {
		AssetExts = AssetExts[:0]
		if err := md.PrimitiveDecode(p, &AssetExts); err != nil {
			return err
		}
	}

	if p, ok := m["i18n_enabled"]; ok {
		if err := md.PrimitiveDecode(p, &I18nEnabled); err != nil {
			return err
		}
	}

	if p, ok := m["locale_root"]; ok {
		if err := md.PrimitiveDecode(p, &LocaleRoot); err != nil {
			return err
		}
	}

	if p, ok := m["locale_base"]; ok {
		if err := md.PrimitiveDecode(p, &LocaleBase); err != nil {
			return err
		}
	}

	return theServer.serve()
}

// Close closes the server immediately.
func Close() error {
	return theServer.close()
}

// Shutdown gracefully shuts down the server without interrupting any active
// connections until timeout. It waits indefinitely for connections to return to
// idle and then shut down when the timeout is less than or equal to zero.
func Shutdown(timeout time.Duration) error {
	return theServer.shutdown(timeout)
}

// Handler defines a function to serve requests.
type Handler func(*Request, *Response) error

// NotFoundHandler is a `Handler` that returns not found error.
var NotFoundHandler = func(req *Request, res *Response) error {
	res.Status = http.StatusNotFound
	return errors.New(http.StatusText(res.Status))
}

// MethodNotAllowedHandler is a `Handler` that returns method not allowed error.
var MethodNotAllowedHandler = func(req *Request, res *Response) error {
	res.Status = http.StatusMethodNotAllowed
	return errors.New(http.StatusText(res.Status))
}

// Gas defines a function to process gases.
type Gas func(Handler) Handler

// WrapHTTPMiddleware provides a convenient way to wrap an `http.Handler`
// middleware into a `Gas`.
func WrapHTTPMiddleware(m func(http.Handler) http.Handler) Gas {
	return func(next Handler) Handler {
		return func(req *Request, res *Response) error {
			var err error
			m(http.HandlerFunc(func(
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
type errorLogWriter struct{}

// Write implements the `io.Writer`.
func (*errorLogWriter) Write(b []byte) (int, error) {
	ERROR(strings.TrimSuffix(string(b), "\n"))
	return len(b), nil
}

// stringsContainsCIly reports whether the ss contains the s case-insensitively.
func stringsContainsCIly(ss []string, s string) bool {
	s = strings.ToLower(s)
	for _, v := range ss {
		if strings.ToLower(v) == s {
			return true
		}
	}

	return false
}

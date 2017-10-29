package air

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// AppName is the name of the current web application.
//
// It is called "app_name" in the "config.toml".
var AppName = "air"

// DebugMode indicates whether the current web application is in debug mode.
//
// It is called "debug_mode" in the "config.toml".
var DebugMode = false

// LoggerEnabled indicates whether the logger is enabled.
//
// It will be forced to be true when the `DebugMode` is true.
//
// It is called "logger_enabled" in the "config.toml".
var LoggerEnabled = false

// LoggerFormat is the output format of the logger.
//
// It is called "logger_format" in the "config.toml".
var LoggerFormat = `{"app_name":"{{.AppName}}","time":"{{.Time}}",` +
	`"level":"{{.Level}}","file":"{{.File}}","line":"{{.Line}}",` +
	`"message":"{{.Message}}"}`

// LoggerOutput is the output destination of the logger.
var LoggerOutput = io.Writer(os.Stdout)

// Address is the TCP address that the HTTP server listens on.
//
// It is called "address" in the "config.toml".
var Address = "localhost:2333"

// ReadTimeout is the maximum duration the HTTP server reads an HTTP request.
//
// It is called "read_timeout" in the "config.toml".
//
// **It is unit in the "config.toml" is MILLISECONDS.**
var ReadTimeout = time.Duration(0)

// ReadHeaderTimeout is the amount of time allowed the HTTP server reads the
// HTTP request headers.
//
// It is called "read_header_timeout" in the "config.toml".
//
// **It is unit in the "config.toml" is MILLISECONDS.**
var ReadHeaderTimeout = time.Duration(0)

// WriteTimeout is the maximum duration the HTTP server writes an HTTP response.
//
// It is called "write_timeout" in the "config.toml".
//
// **It is unit in the "config.toml" is MILLISECONDS.**
var WriteTimeout = time.Duration(0)

// IdleTimeout is the maximum amount of time the HTTP server waits for the next
// HTTP request when HTTP keey-alives are enabled. If it is zero, the value of
// `ReadTimeout` is used. If both are zero, `ReadHeaderTimeout` is used.
//
// It is called "idle_timeout" in the "config.toml".
//
// **It is unit in the "config.toml" is MILLISECONDS.**
var IdleTimeout = time.Duration(0)

// MaxHeaderBytes is the maximum number of bytes the HTTP server will read
// parsing the HTTP request header's keys and values, including the HTTP request
// line.
//
// It is called "max_header_bytes" in the "config.toml".
var MaxHeaderBytes = 1 << 20

// TLSCertFile is the path to the TLS certificate file used when starting the
// HTTP server.
//
// It is called "tls_cert_file" in the "config.toml".
var TLSCertFile = ""

// TLSKeyFile is the path to the TLS key file used when starting the HTTP
// server.
//
// It is called "tls_key_file" in the "config.toml".
var TLSKeyFile = ""

// ErrorHandler is the centralized error handler for the HTTP server.
var ErrorHandler = func(err error, req *Request, res *Response) {
	e := &Error{
		Code:    500,
		Message: "Internal Server Error",
	}
	if ce, ok := err.(*Error); ok {
		e = ce
	} else if DebugMode {
		e.Message = err.Error()
	}
	if !res.Written {
		res.StatusCode = e.Code
		res.String(e.Message)
	}
}

// PreGases is the `Gas` chain that performs first than the HTTP router.
var PreGases = []Gas{}

// Gases is the `Gas` chain that performs after than the HTTP router.
var Gases = []Gas{}

// MinifierEnabled indicates whether the minifier is enabled.
//
// It is called "minifier_enabled" in the "config.toml".
var MinifierEnabled = false

// TemplateRoot is the root of the HTML templates. All the HTTP templates inside
// it will be recursively parsed into the renderer.
//
// It is called "template_root" in the "config.toml".
var TemplateRoot = "templates"

// TemplateExts is the file name extensions of the HTML templates used to
// distinguish the HTTP template files in the `TemplateRoot` when parsing them
// into the renderer.
//
// It is called "template_exts" in the "config.toml".
var TemplateExts = []string{".html"}

// TemplateLeftDelim is the left side of the HTML template delimiter the
// renderer renders the HTTP templates.
//
// It is called "template_left_delim" in the "config.toml".
var TemplateLeftDelim = "{{"

// TemplateRightDelim is the right side of the HTML template delimiter the
// renderer renders the HTTP templates.
//
// It is called "template_right_delim" in the "config.toml".
var TemplateRightDelim = "}}"

// TemplateFuncMap is the HTTP template function map the renderer renders the
// HTTP templates.
var TemplateFuncMap = map[string]interface{}{
	"strlen":  strlen,
	"strcat":  strcat,
	"substr":  substr,
	"timefmt": timefmt,
}

// CofferEnabled indicates whether the coffer is enabled.
//
// It is called "coffer_enabled" in the "config.toml".
var CofferEnabled = false

// AssetRoot represents the root of the asset files. All the asset files inside
// it will be recursively parsed into the coffer.
//
// It is called "asset_root" in the "config.toml".
var AssetRoot = "assets"

// AssetExts is the file name extensions of the asset files used to distinguish
// the asset files in the `AssetRoot` when loading them into the coffer.
//
// It is called "asset_exts" in the "config.toml".
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
}

// Config is the key-value pairs that is parsed from the "config.toml".
var Config = map[string]interface{}{}

func init() {
	b, _ := ioutil.ReadFile("config.toml")
	toml.Unmarshal(b, &Config)
	if an, ok := Config["app_name"].(string); ok {
		AppName = an
	}
	if dm, ok := Config["debug_mode"].(bool); ok {
		DebugMode = dm
	}
	if le, ok := Config["logger_enabled"].(bool); ok {
		LoggerEnabled = le
	}
	if lf, ok := Config["logger_format"].(string); ok {
		LoggerFormat = lf
	}
	if addr, ok := Config["address"].(string); ok {
		Address = addr
	}
	if rt, ok := Config["read_timeout"].(int64); ok {
		ReadTimeout = time.Duration(rt) * time.Millisecond
	}
	if rht, ok := Config["read_header_timeout"].(int64); ok {
		ReadHeaderTimeout = time.Duration(rht) * time.Millisecond
	}
	if wt, ok := Config["write_timeout"].(int64); ok {
		WriteTimeout = time.Duration(wt) * time.Millisecond
	}
	if it, ok := Config["idle_timeout"].(int64); ok {
		IdleTimeout = time.Duration(it) * time.Millisecond
	}
	if mhb, ok := Config["max_header_bytes"].(int64); ok {
		MaxHeaderBytes = int(mhb)
	}
	if tcf, ok := Config["tls_cert_file"].(string); ok {
		TLSCertFile = tcf
	}
	if tkf, ok := Config["tls_key_file"].(string); ok {
		TLSKeyFile = tkf
	}
	if me, ok := Config["minifier_enabled"].(bool); ok {
		MinifierEnabled = me
	}
	if tr, ok := Config["template_root"].(string); ok {
		TemplateRoot = tr
	}
	if tes, ok := Config["template_exts"].([]interface{}); ok {
		TemplateExts = nil
		for _, te := range tes {
			TemplateExts = append(TemplateExts, te.(string))
		}
	}
	if tld, ok := Config["template_left_delim"].(string); ok {
		TemplateLeftDelim = tld
	}
	if trd, ok := Config["template_right_delim"].(string); ok {
		TemplateRightDelim = trd
	}
	if ce, ok := Config["coffer_enabled"].(bool); ok {
		CofferEnabled = ce
	}
	if ar, ok := Config["asset_root"].(string); ok {
		AssetRoot = ar
	}
	if aes, ok := Config["asset_exts"].([]interface{}); ok {
		AssetExts = nil
		for _, ae := range aes {
			AssetExts = append(AssetExts, ae.(string))
		}
	}
}

// Serve starts the HTTP server.
func Serve() error {
	return serverSingleton.serve()
}

// Close closes the HTTP server immediately.
func Close() error {
	return serverSingleton.close()
}

// Shutdown gracefully shuts down the HTTP server without interrupting any
// active connections until timeout. It waits indefinitely for connections to
// return to idle and then shut down when the timeout is less than or equal to
// zero.
func Shutdown(timeout time.Duration) error {
	return serverSingleton.shutdown(timeout)
}

// INFO logs the v at the INFO level.
func INFO(v ...interface{}) {
	loggerSingleton.log("INFO", v...)
}

// WARN logs the v at the WARN level.
func WARN(v ...interface{}) {
	loggerSingleton.log("WARN", v...)
}

// ERROR logs the v at the ERROR level.
func ERROR(v ...interface{}) {
	loggerSingleton.log("ERROR", v...)
}

// PANIC logs the v at the PANIC level.
func PANIC(v ...interface{}) {
	loggerSingleton.log("PANIC", v...)
	panic(fmt.Sprint(v...))
}

// FATAL logs the v at the FATAL level.
func FATAL(v ...interface{}) {
	loggerSingleton.log("FATAL", v...)
	os.Exit(1)
}

// GET registers a new GET route for the path with the matching h in the router
// with the optional route-level gases.
func GET(path string, h Handler, gases ...Gas) {
	routerSingleton.register("GET", path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router with the optional route-level gases.
func HEAD(path string, h Handler, gases ...Gas) {
	routerSingleton.register("HEAD", path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router with the optional route-level gases.
func POST(path string, h Handler, gases ...Gas) {
	routerSingleton.register("POST", path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// with the optional route-level gases.
func PUT(path string, h Handler, gases ...Gas) {
	routerSingleton.register("PUT", path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router with the optional route-level gases.
func PATCH(path string, h Handler, gases ...Gas) {
	routerSingleton.register("PATCH", path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router with the optional route-level gases.
func DELETE(path string, h Handler, gases ...Gas) {
	routerSingleton.register("DELETE", path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router with the optional route-level gases.
func CONNECT(path string, h Handler, gases ...Gas) {
	routerSingleton.register("CONNECT", path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router with the optional route-level gases.
func OPTIONS(path string, h Handler, gases ...Gas) {
	routerSingleton.register("OPTIONS", path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router with the optional route-level gases.
func TRACE(path string, h Handler, gases ...Gas) {
	routerSingleton.register("TRACE", path, h, gases...)
}

// STATIC registers a new route with the path prefix to serve the static files
// from the provided root directory.
func STATIC(prefix, root string) {
	GET(prefix+"*", func(req *Request, res *Response) error {
		err := res.File(filepath.Join(root, req.PathParams["*"]))
		if os.IsNotExist(err) {
			return NotFoundHandler(req, res)
		}
		return err
	})
}

// FILE registers a new route with the path to serve a static file.
func FILE(path, file string) {
	GET(path, func(req *Request, res *Response) error {
		err := res.File(file)
		if os.IsNotExist(err) {
			return NotFoundHandler(req, res)
		}
		return err
	})
}

// Handler defines a function to serve HTTP requests.
type Handler func(*Request, *Response) error

// NotFoundHandler is a `Handler` returns HTTP not found error.
var NotFoundHandler = func(*Request, *Response) error {
	return &Error{
		Code:    404,
		Message: "Not Found",
	}
}

// MethodNotAllowedHandler is a `Handler` returns HTTP method not allowed error.
var MethodNotAllowedHandler = func(*Request, *Response) error {
	return &Error{
		Code:    405,
		Message: "Method Not Allowed",
	}
}

// Gas defines a function to process gases.
type Gas func(Handler) Handler

// WrapGas wraps the h into the `Gas`.
func WrapGas(h Handler) Gas {
	return func(next Handler) Handler {
		return func(req *Request, res *Response) error {
			if err := h(req, res); err != nil {
				return err
			}
			return next(req, res)
		}
	}
}

// Error represents the HTTP error.
type Error struct {
	Code    int
	Message string
}

// Error implements the `error#Error()`.
func (e *Error) Error() string {
	return e.Message
}

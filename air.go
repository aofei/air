package air

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/BurntSushi/toml"
)

// AppName is the name of the current web application.
//
// It's called "app_name" in the config file.
var AppName = "air"

// DebugMode indicates whether to enable the debug mode.
//
// It's called "debug_mode" in the config file.
var DebugMode = false

// LoggerEnabled indicates whether to enable the logger.
//
// It will be forced to the true if the `DebugMode` is true.
//
// It's called "logger_enabled" in the config file.
var LoggerEnabled = false

// LoggerFormat is the format of the output content of the logger.
//
// It's called "logger_format" in the config file.
var LoggerFormat = `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}",` +
	`"level":"{{.level}}","file":"{{.short_file}}","line":"{{.line}}"}`

// LoggerOutput is the format of the output content of the logger.
var LoggerOutput = io.Writer(os.Stdout)

// Address is the TCP address that the HTTP server to listen on.
//
// It's called "address" in the config file.
var Address = "localhost:2333"

// ReadTimeout is the maximum duration before timing out the HTTP server will
// read of an HTTP request.
//
// It's called "read_timeout" in the config file.
//
// **It's unit in the config file is MILLISECONDS.**
var ReadTimeout = time.Duration(0)

// WriteTimeout is the maximum duration before timing out the HTTP server will
// write of an HTTP response.
//
// It's called "write_timeout" in the config file.
//
// **It's unit in the config file is MILLISECONDS.**
var WriteTimeout = time.Duration(0)

// MaxHeaderBytes is the maximum number of bytes the HTTP server will read
// parsing an HTTP request header's keys and values, including the HTTP request
// line. It does not limit the size of the HTTP request body.
//
// It's called "max_header_bytes" in the config file.
var MaxHeaderBytes = 1 << 20

// TLSCertFile is the path of the TLS certificate file that will be used by the
// HTTP server.
//
// It's called "tls_cert_file" in the config file.
var TLSCertFile = ""

// TLSKeyFile is the path of the TLS key file that will be used by the HTTP
// server.
//
// It's called "tls_key_file" in the config file.
var TLSKeyFile = ""

// ErrorHandler is the centralized error handler of the HTTP server.
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

// PreGases is a `Gas` chain which is perform before the HTTP router.
var PreGases = []Gas{}

// Gases is a `Gas` chain which is perform after the HTTP router.
var Gases = []Gas{}

// MinifierEnabled indicates whether to enable the minifier.
//
// It's called "minifier_enabled" in the config file.
var MinifierEnabled = false

// TemplateRoot represents the root directory of the HTML templates. It will be
// parsed into the renderer.
//
// It's called "template_root" in the config file.
var TemplateRoot = "templates"

// TemplateExts represents the file name extensions of the HTML templates. It
// will be used when parsing the HTML templates.
//
// It's called "template_exts" in the config file.
var TemplateExts = []string{".html"}

// TemplateLeftDelim represents the left side of the HTML template delimiter. It
// will be used when parsing the HTML templates.
//
// It's called "template_left_delim" in the config file.
var TemplateLeftDelim = "{{"

// TemplateRightDelim represents the right side of the HTML template delimiter.
// It will be used when parsing the HTML templates.
//
// It's called "template_right_delim" in the config file.
var TemplateRightDelim = "}}"

// CofferEnabled indicates whether to enable the coffer.
//
// It's called "coffer_enabled" in the config file.
var CofferEnabled = false

// AssetRoot represents the root directory of the asset file. It will be loaded
// into the coffer.
//
// It's called "asset_root" in the config file.
var AssetRoot = "assets"

// AssetExts represents the file name extensions of the asset file. It will be
// used when loading the asset file.
//
// It's called "asset_exts" in the config file.
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

// Config is the map that parsing from the config file. You can use it to access
// the values in the config file.
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
	if wt, ok := Config["write_timeout"].(int64); ok {
		WriteTimeout = time.Duration(wt) * time.Millisecond
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
// 0.
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
		err := res.File(path.Join(root, req.PathParams["*"]))
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

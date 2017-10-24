package air

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/BurntSushi/toml"
)

// Air is the top-level framework struct.
type Air struct {
	// AppName is the name of the current web application.
	//
	// The default value is "air".
	//
	// It's called "app_name" in the config files.
	AppName string

	// DebugMode indicates whether to enable the debug mode.
	//
	// The default value is false.
	//
	// It's called "debug_mode" in the config files.
	DebugMode bool

	// Logger is used to log information generated in the runtime.
	Logger *Logger

	// LoggerEnabled indicates whether to enable the `Logger`.
	//
	// It will be forced to the true if the `DebugMode` is true.
	//
	// The default value is false.
	//
	// It's called "logger_enabled" in the config files.
	LoggerEnabled bool

	// LoggerFormat is the format of the output content of the `Logger`.
	//
	// The default value is `{"app_name":"{{.app_name}}","time":"` +
	// `{{.time_rfc3339}}","level":"{{.level}}","file":"` +
	// `{{.short_file}}","line":"{{.line}}"}`
	//
	// It's called "logger_format" in the config files.
	LoggerFormat string

	// Address is the TCP address that the HTTP server to listen on.
	//
	// The default value is "localhost:2333".
	//
	// It's called "address" in the config files.
	Address string

	// ReadTimeout is the maximum duration before timing out the HTTP server
	// will read of an HTTP request.
	//
	// The default value is 0.
	//
	// It's called "read_timeout" in the config files.
	//
	// **It's unit in the config files is MILLISECONDS.**
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out the HTTP
	// server will write of an HTTP response.
	//
	// The default value is 0.
	//
	// It's called "write_timeout" in the config files.
	//
	// **It's unit in the config files is MILLISECONDS.**
	WriteTimeout time.Duration

	// MaxHeaderBytes is the maximum number of bytes the HTTP server will
	// read parsing an HTTP request header's keys and values, including the
	// HTTP request line. It does not limit the size of the HTTP request
	// body.
	//
	// The default value is 1048576.
	//
	// It's called "max_header_bytes" in the config files.
	MaxHeaderBytes int

	// TLSCertFile is the path of the TLS certificate file that will be used
	// by the HTTP server.
	//
	// The default value is "".
	//
	// It's called "tls_cert_file" in the config files.
	TLSCertFile string

	// TLSKeyFile is the path of the TLS key file that will be used by the
	// HTTP server.
	//
	// The default value is "".
	//
	// It's called "tls_key_file" in the config files.
	TLSKeyFile string

	// ErrorHandler is the centralized HTTP error handler of the HTTP
	// server.
	ErrorHandler ErrorHandler

	// PreGases is a `Gas` chain which is perform before the HTTP router.
	PreGases []Gas

	// Gases is a `Gas` chain which is perform after the HTTP router.
	Gases []Gas

	// MinifierEnabled indicates whether to enable the minifier when the
	// HTTP server is started.
	//
	// The default value is false.
	//
	// It's called "minifier_enabled" in the config file.
	MinifierEnabled bool

	// TemplateRoot represents the root directory of the HTML templates. It
	// will be parsed into the renderer.
	//
	// The default value is "templates" that means a subdirectory of the
	// runtime directory.
	//
	// It's called "template_root" in the config file.
	TemplateRoot string

	// TemplateExts represents the file name extensions of the HTML
	// templates. It will be used when parsing the HTML templates.
	//
	// The default value is [".html"].
	//
	// It's called "template_exts" in the config file.
	TemplateExts []string

	// TemplateLeftDelim represents the left side of the HTML template
	// delimiter. It will be used when parsing the HTML templates.
	//
	// The default value is "{{".
	//
	// It's called "template_left_delim" in the config file.
	TemplateLeftDelim string

	// TemplateRightDelim represents the right side of the HTML template
	// delimiter. It will be used when parsing the HTML templates.
	//
	// The default value is "}}".
	//
	// It's called "template_right_delim" in the config file.
	TemplateRightDelim string

	// CofferEnabled indicates whether to enable the coffer when the HTTP
	// server is started.
	//
	// The default value is false.
	//
	// It's called "coffer_enabled" in the config file.
	CofferEnabled bool

	// AssetRoot represents the root directory of the asset files. It will
	// be loaded into the coffer.
	//
	// The default value is "assets" that means a subdirectory of the
	// runtime directory.
	//
	// It's called "asset_root" in the config file.
	AssetRoot string

	// AssetExts represents the file name extensions of the asset files. It
	// will be used when loading the asset files.
	//
	// The default value is [".html", ".css", ".js", ".json", ".xml",
	// ".svg", ".jpg", ".png"].
	//
	// It's called "asset_exts" in the config file.
	AssetExts []string

	// Config is the data that parsing from the config files. You can use it
	// to access the values in the config files.
	//
	// e.g. Config["foobar"] will accesses the value in the config files
	// named "foobar".
	Config map[string]interface{}

	server   *server
	router   *router
	binder   *binder
	minifier *minifier
	renderer *renderer
	coffer   *coffer
}

// New returns a new instance of the `Air`.
func New(configFiles ...string) *Air {
	a := &Air{
		AppName: "air",
		LoggerFormat: `{"app_name":"{{.app_name}}","time":"` +
			`{{.time_rfc3339}}","level":"{{.level}}","file":"` +
			`{{.short_file}}","line":"{{.line}}"}`,
		Address:        "localhost:2333",
		MaxHeaderBytes: 1 << 20,
		ErrorHandler: func(err error, req *Request, res *Response) {
			he := &Error{
				Code:    500,
				Message: "Internal Server Error",
			}
			if che, ok := err.(*Error); ok {
				he = che
			} else if req.air.DebugMode {
				he.Message = err.Error()
			}

			if !res.Written {
				res.StatusCode = he.Code
				res.String(he.Message)
			}

			req.air.Logger.Error(err)
		},
		TemplateRoot:       "templates",
		TemplateExts:       []string{".html"},
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		AssetRoot:          "assets",
		AssetExts: []string{
			".html",
			".css",
			".js",
			".json",
			".xml",
			".svg",
			".jpg",
			".png",
		},
	}

	a.Logger = newLogger(a)
	a.server = newServer(a)
	a.router = newRouter(a)
	a.binder = newBinder()
	a.minifier = newMinifier()
	a.renderer = newRenderer(a)
	a.coffer = newCoffer(a)

	for _, cf := range configFiles {
		b, _ := ioutil.ReadFile(cf)
		toml.Unmarshal(b, &a.Config)
	}

	if an, ok := a.Config["app_name"].(string); ok {
		a.AppName = an
	}
	if dm, ok := a.Config["debug_mode"].(bool); ok {
		a.DebugMode = dm
	}
	if le, ok := a.Config["logger_enabled"].(bool); ok {
		a.LoggerEnabled = le
	}
	if lf, ok := a.Config["logger_format"].(string); ok {
		a.LoggerFormat = lf
	}
	if addr, ok := a.Config["address"].(string); ok {
		a.Address = addr
	}
	if rt, ok := a.Config["read_timeout"].(int64); ok {
		a.ReadTimeout = time.Duration(rt) * time.Millisecond
	}
	if wt, ok := a.Config["write_timeout"].(int64); ok {
		a.WriteTimeout = time.Duration(wt) * time.Millisecond
	}
	if mhb, ok := a.Config["max_header_bytes"].(int64); ok {
		a.MaxHeaderBytes = int(mhb)
	}
	if tcf, ok := a.Config["tls_cert_file"].(string); ok {
		a.TLSCertFile = tcf
	}
	if tkf, ok := a.Config["tls_key_file"].(string); ok {
		a.TLSKeyFile = tkf
	}
	if me, ok := a.Config["minifier_enabled"].(bool); ok {
		a.MinifierEnabled = me
	}
	if tr, ok := a.Config["template_root"].(string); ok {
		a.TemplateRoot = tr
	}
	if tes, ok := a.Config["template_exts"].([]interface{}); ok {
		a.TemplateExts = nil
		for _, te := range tes {
			a.TemplateExts = append(a.TemplateExts, te.(string))
		}
	}
	if tld, ok := a.Config["template_left_delim"].(string); ok {
		a.TemplateLeftDelim = tld
	}
	if trd, ok := a.Config["template_right_delim"].(string); ok {
		a.TemplateRightDelim = trd
	}
	if ce, ok := a.Config["coffer_enabled"].(bool); ok {
		a.CofferEnabled = ce
	}
	if ar, ok := a.Config["asset_root"].(string); ok {
		a.AssetRoot = ar
	}
	if aes, ok := a.Config["asset_exts"].([]interface{}); ok {
		a.AssetExts = nil
		for _, ae := range aes {
			a.AssetExts = append(a.AssetExts, ae.(string))
		}
	}

	return a
}

// Serve starts the HTTP server.
func (a *Air) Serve() error {
	return a.server.serve()
}

// Close closes the HTTP server immediately.
func (a *Air) Close() error {
	return a.server.close()
}

// Shutdown gracefully shuts down the HTTP server without interrupting any
// active connections until timeout. It waits indefinitely for connections to
// return to idle and then shut down when the timeout is negative.
func (a *Air) Shutdown(timeout time.Duration) error {
	return a.server.shutdown(timeout)
}

// GET registers a new GET route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.add("GET", path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) HEAD(path string, h Handler, gases ...Gas) {
	a.add("HEAD", path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.add("POST", path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// with the optional route-level gases.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.add("PUT", path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) PATCH(path string, h Handler, gases ...Gas) {
	a.add("PATCH", path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.add("DELETE", path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) CONNECT(path string, h Handler, gases ...Gas) {
	a.add("CONNECT", path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) OPTIONS(path string, h Handler, gases ...Gas) {
	a.add("OPTIONS", path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router with the optional route-level gases.
func (a *Air) TRACE(path string, h Handler, gases ...Gas) {
	a.add("TRACE", path, h, gases...)
}

// Static registers a new route with the path prefix to serve the static files
// from the provided root directory.
func (a *Air) Static(prefix, root string) {
	a.GET(prefix+"*", func(req *Request, res *Response) error {
		err := res.File(path.Join(root, req.PathParams["*"]))
		if os.IsNotExist(err) {
			return NotFoundHandler(req, res)
		}
		return err
	})
}

// File registers a new route with the path to serve a static file.
func (a *Air) File(path, file string) {
	a.GET(path, func(req *Request, res *Response) error {
		err := res.File(file)
		if os.IsNotExist(err) {
			return NotFoundHandler(req, res)
		}
		return err
	})
}

// add registers a new route for the path with the method and the matching h in
// the router with the optional route-level gases.
func (a *Air) add(method, path string, h Handler, gases ...Gas) {
	a.router.add(method, path, func(req *Request, res *Response) error {
		h := h
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}
		return h(req, res)
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

// Error represents an error that occurred while handling HTTP requests.
type Error struct {
	Code    int
	Message string
}

// Error implements the `error#Error()`.
func (e *Error) Error() string {
	return e.Message
}

// ErrorHandler is a centralized HTTP error handler.
type ErrorHandler func(error, *Request, *Response)

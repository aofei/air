package air

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
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
	"github.com/mitchellh/mapstructure"
	yaml "gopkg.in/yaml.v2"
)

// Air is the top-level struct of this framework. Please keep in mind that the
// new instances should only be created by calling the `New()`. If you only need
// one instance, then it is recommended to use the `Default`, which will help
// you simplify the scope management.
type Air struct {
	// AppName is the name of the current web application. It is recommended
	// to set this field and try to ensure that the value of this field is
	// unique (used to distinguish between different web applications).
	//
	// Default value: "air"
	AppName string `mapstructure:"app_name"`

	// MaintainerEmail is the e-mail address of the one who is responsible
	// for maintaining the current web application. It is recommended to set
	// this field if the ACME is enabled (used by CAs, such as Let's
	// Encrypt, to notify about problems with issued certificates).
	//
	// Default value: ""
	MaintainerEmail string `mapstructure:"maintainer_email"`

	// DebugMode indicates whether the current web application is in debug
	// mode. Please keep in mind that this field is quite bossy and should
	// never be used in non-debug mode.
	//
	// ATTENTION: Some features will be affected in debug mode.
	//
	// Default value: false
	DebugMode bool `mapstructure:"debug_mode"`

	// Address is the TCP address that the server of the current web
	// application listens on. This field is never empty and must contain
	// a port part.
	//
	// Default value: ":8080"
	Address string `mapstructure:"address"`

	// HostWhitelist is the hosts allowed by the server of the current web
	// application. It is highly recommended to set this field when in
	// non-debug mode. If the length of this field is not zero, then all
	// connections that are not connected to the hosts in this field will be
	// rejected.
	//
	// It only works when the `DebugMode` is false.
	//
	// Default value: nil
	HostWhitelist []string `mapstructure:"host_whitelist"`

	// ReadTimeout is the maximum duration the server of the current web
	// application reads a request, including the body part. It does not let
	// the handlers make per-request decisions on each request body's
	// acceptable deadline or upload rate.
	//
	// Default value: 0
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed the server of the
	// current web application reads the headers of a request. The
	// connection's read deadline is reset after reading the headers and the
	// handler can decide what is considered too slow for the body.
	//
	// Default value: 0
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`

	// WriteTimeout is the maximum duration the server of the current web
	// application writes a response. It is reset whenever a new request's
	// header is read. Like `ReadTimeout`, it does not let handlers make
	// decisions on a per-request basis.
	//
	// Default value: 0
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time the server of the current
	// web application waits for the next request. If it is zero, the value
	// of `ReadTimeout` is used. If both are zero, the value of
	// `ReadHeaderTimeout` is used.
	//
	// Default value: 0
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// MaxHeaderBytes is the maximum number of bytes the server of the
	// current web application will read parsing the request header's names
	// and values, including the request line.
	//
	// Default value: 1048576
	MaxHeaderBytes int `mapstructure:"max_header_bytes"`

	// TLSCertFile is the path to the TLS certificate file used when
	// starting the server of the current web application. If the
	// certificate is signed by a certificate authority, the TLS certificate
	// file should be the concatenation of the certificate, any
	// intermediates, and the CA's certificate.
	//
	// ATTENTION: This field must be set at the same time as the
	// `TLSKeyFile` to make the server of the current web application to
	// handle requests on incoming TLS connections.
	//
	// Default value: ""
	TLSCertFile string `mapstructure:"tls_cert_file"`

	// TLSKeyFile is the path to the TLS key file used when starting the
	// server of the current web application. The TLS key file must match
	// the TLS certificate targeted by the `TLSCertFile`.
	//
	// ATTENTION: This field must be set at the same time as the
	// `TLSCertFile` to make the server of the current web application to
	// handle requests on incoming TLS connections.
	//
	// Default value: ""
	TLSKeyFile string `mapstructure:"tls_key_file"`

	// ACMEEnabled indicates whether the ACME feature of the current web
	// application is enabled. This feature gives the server of the current
	// web application the ability to automatically retrieve new TLS
	// certificates from the ACME CA.
	//
	// It only works when the `DebugMode` is false and both of the
	// `TLSCertFile` and the `TLSKeyFile` are empty.
	//
	// Default value: false
	ACMEEnabled bool `mapstructure:"acme_enabled"`

	// ACMEDirectoryURL is the CA directory URL of the ACME feature of the
	// current web application. This CA directory must be trusted because
	// the ACME will automatically accept the Terms of Service (TOS)
	// prompted in it.
	//
	// Default value: "https://acme-v01.api.letsencrypt.org/directory"
	ACMEDirectoryURL string `mapstructure:"acme_directory_url"`

	// ACMECertRoot is the root of the certificates of the ACME feature of
	// the current web application. It is recommended to set this field to a
	// persistent place. By the way, different web applications can share
	// the same place (if they are all built using this framework).
	//
	// Default value: "acme-certs"
	ACMECertRoot string `mapstructure:"acme_cert_root"`

	// HTTPSEnforced indicates whether the current web application is
	// forcibly accessible only via the HTTPS scheme (HTTP requests will
	// automatically redirect to HTTPS).
	//
	// It only works when the port of the `Address` is neither "80" nor
	// "http" and the server of the current web application can handle
	// requests on incoming TLS connections.
	//
	// Default value: false
	HTTPSEnforced bool `mapstructure:"https_enforced"`

	// WebSocketHandshakeTimeout is the maximum amount of time the server of
	// the current web application waits for a WebSocket handshake to
	// complete.
	//
	// Default value: 0
	WebSocketHandshakeTimeout time.Duration `mapstructure:"websocket_handshake_timeout"`

	// WebSocketSubprotocols is the supported WebSocket subprotocols of the
	// server of the current web application. If the length of this field is
	// not zero, then the `Response#WebSocket()` negotiates a subprotocol by
	// selecting the first match in this field with a protocol requested by
	// the client. If there is no match, then no protocol is negotiated (the
	// "Sec-Websocket-Protocol" header is not included in the handshake
	// response).
	//
	// Default value: nil
	WebSocketSubprotocols []string `mapstructure:"websocket_subprotocols"`

	// Pregases is the `Gas` chain stack of the current web application
	// that performs before routing.
	//
	// Default value: nil
	Pregases []Gas `mapstructure:"-"`

	// Gases is the `Gas` chain stack of the current web application that
	// performs after routing.
	//
	// Default value: nil
	Gases []Gas `mapstructure:"-"`

	// NotFoundHandler is the `Handler` of the current web application that
	// returns not found error. This field is never nil because the router
	// of the current web application will use it as the default `Handler`
	// when no matching routes are found.
	//
	// Default value: `DefaultNotFoundHandler`
	NotFoundHandler func(*Request, *Response) error `mapstructure:"-"`

	// MethodNotAllowedHandler is the `Handler` of the current web
	// application that returns method not allowed error. This field is
	// never nil because the router of the current web application will use
	// it as the default `Handler` when it finds a matching route but its
	// method is not registered.
	//
	// Default value: `DefaultMethodNotAllowedHandler`
	MethodNotAllowedHandler func(*Request, *Response) error `mapstructure:"-"`

	// ErrorHandler is the centralized error handler of the server of the
	// current web application. This field is never nil because it is used
	// in every request-response cycle that has an error.
	//
	// Default value: `DefaultErrorHandler`
	ErrorHandler func(error, *Request, *Response) `mapstructure:"-"`

	// ErrorLogger is the `log.Logger` that logs errors that occur in the
	// server of the current web application. If nil, logging is done via
	// the log package's standard logger.
	//
	// Default value: nil
	ErrorLogger *log.Logger `mapstructure:"-"`

	// AutoPushEnabled indicates whether the HTTP/2 server push automatic
	// mechanism feature of the current web application is enabled. This
	// feature gives the `Response#WriteHTML()` the ability to automatically
	// analyze the response content and use the `Response#Push()` to push
	// the appropriate resources to the client.
	//
	// It only works when the protocol version of the request is HTTP/2.
	//
	// Default value: false
	AutoPushEnabled bool `mapstructure:"auto_push_enabled"`

	// MinifierEnabled indicates whether the minifier feature of the current
	// web application is enabled. This feature gives the `Response#Write()`
	// the ability to minify the matching response content based on the
	// "Content-Type" header.
	//
	// Default value: false
	MinifierEnabled bool `mapstructure:"minifier_enabled"`

	// MinifierMIMETypes is the MIME types of the minifier feature of the
	// current web application that will be minified. Unsupported MIME types
	// will be silently ignored.
	//
	// Default value: ["text/html", "text/css", "application/javascript",
	// "application/json", "application/xml", "image/svg+xml"]
	MinifierMIMETypes []string `mapstructure:"minifier_mime_types"`

	// GzipEnabled indicates whether the gzip feature of the current web
	// application is enabled. This feature gives the `Response` the ability
	// to gzip the matching response content based on the "Content-Type"
	// header.
	//
	// Default value: false
	GzipEnabled bool `mapstructure:"gzip_enabled"`

	// GzipMIMETypes is the MIME types of the gzip feature of the current
	// web application that will be gzipped.
	//
	// Default value: ["text/plain", "text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "application/toml", "application/yaml", "image/svg+xml"]
	GzipMIMETypes []string `mapstructure:"gzip_mime_types"`

	// GzipCompressionLevel is the compression level of the gzip feature of
	// the current web application.
	//
	// Default value: `gzip.DefaultCompression`
	GzipCompressionLevel int `mapstructure:"gzip_compression_level"`

	// RendererTemplateRoot is the root of the HTML templates of the
	// renderer feature of the current web application. All the HTML
	// template files inside it will be recursively parsed into the
	// renderer.
	//
	// Default value: "templates"
	RendererTemplateRoot string `mapstructure:"renderer_template_root"`

	// RendererTemplateExts is the filename extensions of the HTML templates
	// of the renderer feature of the current web application used to
	// distinguish the HTML template files in the `RendererTemplateRoot`
	// when parsing them into the renderer.
	//
	// Default value: [".html"]
	RendererTemplateExts []string `mapstructure:"renderer_template_exts"`

	// RendererTemplateLeftDelim is the left side of the HTML template
	// delimiter of the renderer feature of the current web application.
	//
	// default value: "{{"
	RendererTemplateLeftDelim string `mapstructure:"renderer_template_left_delim"`

	// RendererTemplateRightDelim is the right side of the HTML template
	// delimiter of the renderer feature of the current web application.
	//
	// Default value: "}}"
	RendererTemplateRightDelim string `mapstructure:"renderer_template_right_delim"`

	// RendererTemplateFuncMap is the HTML template function map of the
	// renderer feature of the current web application.
	//
	// Default value: {"strlen": strlen, "substr": substr, "timefmt":
	// timefmt}
	RendererTemplateFuncMap template.FuncMap `mapstructure:"-"`

	// CofferEnabled indicates whether the coffer feature of the current web
	// application is enabled. This feature gives the `Response#WriteFile()`
	// the ability to use the runtime memory to reduce disk I/O pressure.
	//
	// Default value: false
	CofferEnabled bool `mapstructure:"coffer_enabled"`

	// CofferMaxMemoryBytes is the maximum number of bytes of the runtime
	// memory the coffer feature of the current web application will use.
	//
	// Default value: 33554432
	CofferMaxMemoryBytes int `mapstructure:"coffer_max_memory_bytes"`

	// CofferAssetRoot is the root of the asset files of the coffer feature
	// of the current web application. All the asset files inside it will be
	// recursively parsed into the coffer.
	//
	// Default value: "assets"
	CofferAssetRoot string `mapstructure:"coffer_asset_root"`

	// CofferAssetExts is the filename extensions of the asset files used to
	// distinguish the asset files in the `AssetRoot` when loading them into
	// the coffer.
	//
	// Default value: [".html", ".css", ".js", ".json", ".xml", ".toml",
	// ".yaml", ".yml", ".svg", ".jpg", ".jpeg", ".png", ".gif"]
	CofferAssetExts []string `mapstructure:"coffer_asset_exts"`

	// I18nEnabled indicates whether the i18n feature of the current web
	// application is enabled. This feature gives the
	// `Request#LocalizedString()` and the `Response#Render()` the ability
	// to adapt to the request's favorite conventions.
	//
	// Default value: false
	I18nEnabled bool `mapstructure:"i18n_enabled"`

	// I18nLocaleRoot is the root of the locale files of the i18n feature of
	// the current web application. All the locale files inside it will be
	// parsed into the i18n.
	//
	// Default value: "locales"
	I18nLocaleRoot string `mapstructure:"i18n_locale_root"`

	// I18nLocaleBase is the base of the locale files of the i18n feature of
	// the current web application. It will be used when a locale file
	// cannot be found.
	//
	// Default value: "en-US"
	I18nLocaleBase string `mapstructure:"i18n_locale_base"`

	// ConfigFile is the path to the configuration file that will be parsed
	// into the matching fields of the current web application before
	// starting the server.
	//
	// The ".json" extension represents the configuration file is
	// JSON-based.
	//
	// The ".xml" extension represents the configuration file is XML-based.
	//
	// The ".toml" extension represents the configuration file is
	// TOML-based.
	//
	// The ".yaml" and the ".yml" extensions represents the configuration
	// file is YAML-based.
	//
	// Default value: ""
	ConfigFile string `mapstructure:"-"`

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
			"application/toml",
			"application/yaml",
			"image/svg+xml",
		},
		RendererTemplateRoot:       "templates",
		RendererTemplateExts:       []string{".html"},
		RendererTemplateLeftDelim:  "{{",
		RendererTemplateRightDelim: "}}",
		RendererTemplateFuncMap: template.FuncMap{
			"strlen":  strlen,
			"substr":  substr,
			"timefmt": timefmt,
		},
		CofferMaxMemoryBytes: 32 << 20,
		CofferAssetRoot:      "assets",
		CofferAssetExts: []string{
			".html",
			".css",
			".js",
			".json",
			".xml",
			".toml",
			".yaml",
			".yml",
			".svg",
			".jpg",
			".jpeg",
			".png",
			".gif",
		},
		I18nLocaleRoot: "locales",
		I18nLocaleBase: "en-US",
	}

	a.errorLogger = log.New(newErrorLogWriter(a), "", 0)
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

		m := map[string]interface{}{}
		switch e := strings.ToLower(filepath.Ext(a.ConfigFile)); e {
		case ".json":
			err = json.Unmarshal(b, &m)
		case ".xml":
			err = xml.Unmarshal(b, &m)
		case ".toml":
			err = toml.Unmarshal(b, &m)
		case ".yaml", ".yml":
			err = yaml.Unmarshal(b, &m)
		default:
			err = fmt.Errorf(
				"air: unsupported configuration file "+
					"extension %q",
				e,
			)
		}

		if err != nil {
			return err
		} else if err := mapstructure.Decode(m, a); err != nil {
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
	if !strings.HasPrefix(s, "air: ") {
		s = "air: " + s
	}

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

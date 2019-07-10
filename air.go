/*
Package air implements an ideally refined web framework for Go.

Router

A router is basically the most important component of a web framework. In this
framework, registering a route usually requires at least two params:

	air.Default.GET(
		"/users/:UserID/posts/:PostID/assets/*",
		func(req *air.Request, res *air.Response) error {
			userID, err := req.Param("UserID").Value().Int64()
			if err != nil {
				return err
			}

			postID, err := req.Param("PostID").Value().Int64()
			if err != nil {
				return err
			}

			assetPath := req.Param("*").Value().String()

			return res.WriteJSON(map[string]interface{}{
				"user_id":    userID,
				"post_id":    postID,
				"asset_path": assetPath,
			})
		},
	)

The first param is a route path that contains 6 components. Among them, "users",
"posts" and "assets" are static components, ":UserID" and ":PostID" are param
components, "*" is an any param component. Note that all route params (param
component(s) and any param component) will be parsed into the `Request` and can
be accessed via the `Request.Param` and the `Request.Params`. The name of a
`RequestParam` parsed from a param component always discards its leading ":",
such as ":UserID" will become "UserID". The name of a `RequestParam` parsed from
an any param component is "*".

The second param is a `Handler` that serves the requests that match this route.
*/
package air

import (
	"compress/gzip"
	"context"
	"encoding/json"
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

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/mapstructure"
	yaml "gopkg.in/yaml.v2"
)

// Air is the top-level struct of this framework.
//
// It is highly recommended not to modify the value of any field of the `Air`
// after calling the `Air.Serve`, which will cause unpredictable problems.
//
// The new instances of the `Air` should only be created by calling the `New`.
// If you only need one instance of the `Air`, then it is recommended to use the
// `Default`, which will help you simplify the scope management.
type Air struct {
	// AppName is the name of the current web application.
	//
	// It is recommended to set a name and try to ensure that the name is
	// unique (used to distinguish between different web applications).
	//
	// Default value: "air"
	AppName string `mapstructure:"app_name"`

	// MaintainerEmail is the e-mail address of the one who is responsible
	// for maintaining the current web application.
	//
	// It is recommended to set an e-mail if the ACME feature of the current
	// web application is enabled (used by the CAs, such as Let's Encrypt,
	// to notify about problems with issued certificates).
	//
	// Default value: ""
	MaintainerEmail string `mapstructure:"maintainer_email"`

	// DebugMode indicates whether the current web application is in debug
	// mode.
	//
	// Please keep in mind that the debug mode is quite bossy, some features
	// of the current web application will be affected in the debug mode. So
	// never use the debug mode in a production environment unless you want
	// to do something crazy.
	//
	// Default value: false
	DebugMode bool `mapstructure:"debug_mode"`

	// Address is the TCP address that the server of the current web
	// application listens on.
	//
	// There is always an address here that contains a free port.
	//
	// If the port of the `Address` is "0", a port is automatically chosen.
	// The `Addresses` can be used to discover the chosen port.
	//
	// Default value: ":8080"
	Address string `mapstructure:"address"`

	// ReadTimeout is the maximum duration the server of the current web
	// application reads a request entirely, including the body part.
	//
	// The `ReadTimeout` does not let the handlers make per-request
	// decisions on each request body's acceptable deadline or upload rate.
	//
	// Default value: 0
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed the server of the
	// current web application reads the headers of a request.
	//
	// The connection's read deadline is reset after reading the headers and
	// the handler can decide what is considered too slow for the body.
	//
	// Default value: 0
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`

	// WriteTimeout is the maximum duration the server of the current web
	// application writes a response.
	//
	// The `WriteTimeout` is reset whenever a new request's header is read.
	// Like the `ReadTimeout`, the `WriteTimeout` does not let handlers make
	// decisions on a per-request basis.
	//
	// Default value: 0
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time the server of the current
	// web application waits for the next request.
	//
	// If the `IdleTimeout` is zero, the value of the `ReadTimeout` is used.
	// If both are zero, the value of the `ReadHeaderTimeout` is used.
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
	// starting the server of the current web application.
	//
	// If the certificate is signed by a certificate authority, the TLS
	// certificate file should be the concatenation of the certificate, any
	// intermediates, and the CA's certificate.
	//
	// The `TLSCertFile` must be set at the same time as the `TLSKeyFile` to
	// make the server of the current web application to handle requests on
	// incoming TLS connections.
	//
	// Default value: ""
	TLSCertFile string `mapstructure:"tls_cert_file"`

	// TLSKeyFile is the path to the TLS key file used when starting the
	// server of the current web application.
	//
	// The key must match the certificate targeted by the `TLSCertFile`.
	//
	// The `TLSKeyFile` must be set at the same time as the `TLSCertFile` to
	// make the server of the current web application to handle requests on
	// incoming TLS connections.
	//
	// Default value: ""
	TLSKeyFile string `mapstructure:"tls_key_file"`

	// ACMEEnabled indicates whether the ACME feature of the current web
	// application is enabled.
	//
	// The `ACMEEnabled` gives the server of the current web application the
	// ability to automatically retrieve new TLS certificates from the ACME
	// CA targeted by the `ACMEDirectoryURL`.
	//
	// The `ACMEEnabled` only works when the `DebugMode` is false and both
	// the `TLSCertFile` and the `TLSKeyFile` are empty.
	//
	// Default value: false
	ACMEEnabled bool `mapstructure:"acme_enabled"`

	// ACMEDirectoryURL is the CA directory URL of the ACME feature of the
	// current web application.
	//
	// The CA directory must be trusted because the ACME will automatically
	// accept the Terms of Service (TOS) prompted from it.
	//
	// Default value: "https://acme-v01.api.letsencrypt.org/directory"
	ACMEDirectoryURL string `mapstructure:"acme_directory_url"`

	// ACMECertRoot is the root of the certificates of the ACME feature of
	// the current web application.
	//
	// It is recommended to set a persistent root since all CAs have a rate
	// limit on issuing certificates. Different web applications can share
	// the same place (if they are all built using this framework).
	//
	// Default value: "acme-certs"
	ACMECertRoot string `mapstructure:"acme_cert_root"`

	// ACMEHostWhitelist is the list of hosts allowed by the ACME feature of
	// the current web application.
	//
	// It is highly recommended to set a list of hosts. If the length of the
	// list is not zero, then all connections that are not connected to the
	// hosts in the list will not be able to obtain new TLS certificates
	// from the ACME CA targeted by the `ACMEDirectoryURL`.
	//
	// Default value: nil
	ACMEHostWhitelist []string `mapstructure:"acme_host_whitelist"`

	// HTTPSEnforced indicates whether the current web application is
	// forcibly accessible only via the HTTPS scheme (HTTP requests will
	// automatically redirect to HTTPS).
	//
	// The `HTTPSEnforced` only works when the server of the current web
	// application can handle requests on incoming TLS connections.
	//
	// The `HTTPSEnforced` will be forced to true when the `ACMEEnabled` is
	// true, the `DebugMode` is false and both the `TLSCertFile` and the
	// `TLSKeyFile` are empty.
	//
	// Default value: false
	HTTPSEnforced bool `mapstructure:"https_enforced"`

	// HTTPSEnforcedPort is the port of the TCP address (share the same host
	// as the `Address`) that the server of the current web application
	// listens on. All requests to this port will be forced to redirect to
	// HTTPS.
	//
	// If the `HTTPSEnforcedPort` is "0", a port is automatically chosen.
	// The `Addresses` can be used to discover the chosen port.
	//
	// Default value: "80"
	HTTPSEnforcedPort string `mapstructure:"https_enforced_port"`

	// WebSocketHandshakeTimeout is the maximum amount of time the server of
	// the current web application waits for a WebSocket handshake to
	// complete.
	//
	// Default value: 0
	WebSocketHandshakeTimeout time.Duration `mapstructure:"websocket_handshake_timeout"`

	// WebSocketSubprotocols is the list of supported WebSocket subprotocols
	// of the server of the current web application.
	//
	// If the length of the list is not zero, then the `Response.WebSocket`
	// negotiates a subprotocol by selecting the first match in the list
	// with a protocol requested by the client. If there is no match, then
	// no protocol is negotiated (the Sec-Websocket-Protocol header is not
	// included in the handshake response).
	//
	// Default value: nil
	WebSocketSubprotocols []string `mapstructure:"websocket_subprotocols"`

	// PROXYProtocolEnabled indicates whether the PROXY protocol feature of
	// the current web application is enabled.
	//
	// The `PROXYProtocolEnabled` gives the server of the current web
	// application the ability to support the PROXY protocol (See
	// https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt).
	//
	// Default value: false
	PROXYProtocolEnabled bool `mapstructure:"proxy_protocol_enabled"`

	// PROXYProtocolReadHeaderTimeout is the amount of time allowed the
	// PROXY protocol feature of the current web application reads the
	// PROXY protocol header of a connection.
	//
	// The connection's read deadline is reset after reading the PROXY
	// protocol header.
	//
	// Default value: 0
	PROXYProtocolReadHeaderTimeout time.Duration `mapstructure:"proxy_protocol_read_header_timeout"`

	// PROXYProtocolRelayerIPWhitelist is the list of IP addresses or CIDR
	// notation IP address ranges of the relayers allowed by the PROXY
	// protocol feature of the current web application.
	//
	// It is highly recommended to set a list of IP addresses or CIDR
	// notation IP address ranges. If the length of the list is not zero,
	// then all connections relayed from the IP addresses are not in the
	// list will not be able to act the PROXY protocol.
	//
	// Default value: nil
	PROXYProtocolRelayerIPWhitelist []string `mapstructure:"proxy_protocol_relayer_ip_whitelist"`

	// Pregases is the `Gas` chain stack of the current web application
	// that performs before routing.
	//
	// The stack is always FILO.
	//
	// Default value: nil
	Pregases []Gas `mapstructure:"-"`

	// Gases is the `Gas` chain stack of the current web application that
	// performs after routing.
	//
	// The stack is always FILO.
	//
	// Default value: nil
	Gases []Gas `mapstructure:"-"`

	// NotFoundHandler is the `Handler` of the current web application that
	// returns not found error.
	//
	// The `NotFoundHandler` is never nil because the router of the current
	// web application will use it as the default `Handler` when no matching
	// routes are found.
	//
	// Default value: `DefaultNotFoundHandler`
	NotFoundHandler func(*Request, *Response) error `mapstructure:"-"`

	// MethodNotAllowedHandler is the `Handler` of the current web
	// application that returns method not allowed error.
	//
	// The `MethodNotAllowedHandler` is never nil because the router of the
	// current web application will use it as the default `Handler` when it
	// finds a matching route but its method is not registered.
	//
	// Default value: `DefaultMethodNotAllowedHandler`
	MethodNotAllowedHandler func(*Request, *Response) error `mapstructure:"-"`

	// ErrorHandler is the centralized error handler of the server of the
	// current web application.
	//
	// The `ErrorHandler` is never nil because it is used in every
	// request-response cycle that has an error.
	//
	// Default value: `DefaultErrorHandler`
	ErrorHandler func(error, *Request, *Response) `mapstructure:"-"`

	// ErrorLogger is the `log.Logger` that logs errors that occur in the
	// server of the current web application.
	//
	// If the `ErrorLogger` is nil, logging is done via the log package's
	// standard logger.
	//
	// Default value: nil
	ErrorLogger *log.Logger `mapstructure:"-"`

	// AutoPushEnabled indicates whether the HTTP/2 server push automatic
	// mechanism feature of the current web application is enabled.
	//
	// The `AutoPushEnabled` gives the `Response.WriteHTML` the ability to
	// automatically analyze the response content and use the
	// `Response.Push` to push the appropriate resources to the client.
	//
	// The `AutoPushEnabled` only works when the protocol version of the
	// request is HTTP/2.
	//
	// Default value: false
	AutoPushEnabled bool `mapstructure:"auto_push_enabled"`

	// MinifierEnabled indicates whether the minifier feature of the current
	// web application is enabled.
	//
	// The `MinifierEnabled` gives the `Response.Write` the ability to
	// minify the matching response content on the fly based on the
	// Content-Type header.
	//
	// Default value: false
	MinifierEnabled bool `mapstructure:"minifier_enabled"`

	// MinifierMIMETypes is the list of MIME types of the minifier feature
	// of the current web application that will trigger the minimization.
	//
	// Supported MIME types:
	//   * text/html
	//   * text/css
	//   * application/javascript
	//   * application/json
	//   * application/xml
	//   * image/svg+xml
	//
	// Unsupported MIME types will be silently ignored.
	//
	// Default value: ["text/html", "text/css", "application/javascript",
	// "application/json", "application/xml", "image/svg+xml"]
	MinifierMIMETypes []string `mapstructure:"minifier_mime_types"`

	// GzipEnabled indicates whether the gzip feature of the current web
	// application is enabled.
	//
	// The `GzipEnabled` gives the `Response` the ability to gzip the
	// matching response content on the fly based on the Content-Type
	// header.
	//
	// Default value: false
	GzipEnabled bool `mapstructure:"gzip_enabled"`

	// GzipMinContentLength is the minimum content length of the gzip
	// featrue of the current web application used to limit at least how
	// much response content can be gzipped.
	//
	// The content length is determined only from the Content-Length header.
	//
	// Default value: 1024
	GzipMinContentLength int64 `mapstructure:"gzip_min_content_length"`

	// GzipMIMETypes is the list of MIME types of the gzip feature of the
	// current web application that will trigger the gzip.
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

	// GzipFlushThreshold is the flush threshold of the gzip feature of the
	// current web application.
	//
	// Once the pending compressed data in the gzip writer reach the flush
	// threshold, they will be flushed into the underlying writer of the
	// gzip writer immediately.
	//
	// The `GzipFlushThreshold` only works when it is greater than zero.
	//
	// Default value: 8192
	GzipFlushThreshold int `mapstructure:"gzip_flush_threshold"`

	// RendererTemplateRoot is the root of the HTML templates of the
	// renderer feature of the current web application.
	//
	// All HTML template files inside the root will be recursively parsed
	// into the renderer.
	//
	// Default value: "templates"
	RendererTemplateRoot string `mapstructure:"renderer_template_root"`

	// RendererTemplateExts is the list of filename extensions of the HTML
	// templates of the renderer feature of the current web application used
	// to distinguish the HTML template files in the `RendererTemplateRoot`
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
	// Default value: nil
	RendererTemplateFuncMap template.FuncMap `mapstructure:"-"`

	// CofferEnabled indicates whether the coffer feature of the current web
	// application is enabled.
	//
	// The `CofferEnabled` gives the `Response.WriteFile` the ability to use
	// the runtime memory to reduce the disk I/O pressure.
	//
	// Default value: false
	CofferEnabled bool `mapstructure:"coffer_enabled"`

	// CofferMaxMemoryBytes is the maximum number of bytes of the runtime
	// memory the coffer feature of the current web application will use.
	//
	// Default value: 33554432
	CofferMaxMemoryBytes int `mapstructure:"coffer_max_memory_bytes"`

	// CofferAssetRoot is the root of the assets of the coffer feature of
	// the current web application.
	//
	// All asset files inside the root will be recursively parsed into the
	// coffer.
	//
	// Default value: "assets"
	CofferAssetRoot string `mapstructure:"coffer_asset_root"`

	// CofferAssetExts is the list of filename extensions of the assets of
	// the coffer feature of the current web application used to distinguish
	// the asset files in the `CofferAssetRoot` when loading them into the
	// coffer.
	//
	// Default value: [".html", ".css", ".js", ".json", ".xml", ".toml",
	// ".yaml", ".yml", ".svg", ".jpg", ".jpeg", ".png", ".gif"]
	CofferAssetExts []string `mapstructure:"coffer_asset_exts"`

	// I18nEnabled indicates whether the i18n feature of the current web
	// application is enabled.
	//
	// The `I18nEnabled` gives the `Request.LocalizedString` and the
	// `Response.Render` the ability to adapt to the request's favorite
	// conventions based on the Accept-Language header.
	//
	// Default value: false
	I18nEnabled bool `mapstructure:"i18n_enabled"`

	// I18nLocaleRoot is the root of the locales of the i18n feature of the
	// current web application.
	//
	// All TOML-based locale files (".toml" is the extension) inside the
	// root will be parsed into the i18n and their names (without extension)
	// will be used as locales.
	//
	// Default value: "locales"
	I18nLocaleRoot string `mapstructure:"i18n_locale_root"`

	// I18nLocaleBase is the base of the locales of the i18n feature of the
	// current web application used when a locale cannot be found.
	//
	// Default value: "en-US"
	I18nLocaleBase string `mapstructure:"i18n_locale_base"`

	// ConfigFile is the path to the configuration file that will be parsed
	// into the matching fields of the current web application before
	// starting the server.
	//
	// The ".json" extension means the configuration file is JSON-based.
	//
	// The ".toml" extension means the configuration file is TOML-based.
	//
	// The ".yaml" and the ".yml" extensions means the configuration file is
	// YAML-based.
	//
	// Default value: ""
	ConfigFile string `mapstructure:"-"`

	server                       *server
	router                       *router
	binder                       *binder
	minifier                     *minifier
	renderer                     *renderer
	coffer                       *coffer
	i18n                         *i18n
	contentTypeSnifferBufferPool *sync.Pool
	gzipWriterPool               *sync.Pool
	reverseProxyTransport        *http.Transport
	reverseProxyBufferPool       *reverseProxyBufferPool
}

// Default is the default instance of the `Air`.
//
// If you only need one instance of the `Air`, then you should use the
// `Default`. Unless you think you can efficiently pass your instance in
// different scopes.
var Default = New()

// New returns a new instance of the `Air` with default field values.
//
// The `New` is the only function that creates new instances of the `Air` and
// keeps everything working.
func New() *Air {
	a := &Air{
		AppName:                 "air",
		Address:                 ":8080",
		MaxHeaderBytes:          1 << 20,
		ACMEDirectoryURL:        "https://acme-v01.api.letsencrypt.org/directory",
		ACMECertRoot:            "acme-certs",
		HTTPSEnforcedPort:       "80",
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
		GzipMinContentLength: 1 << 10,
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
		GzipCompressionLevel:       gzip.DefaultCompression,
		GzipFlushThreshold:         8 << 10,
		RendererTemplateRoot:       "templates",
		RendererTemplateExts:       []string{".html"},
		RendererTemplateLeftDelim:  "{{",
		RendererTemplateRightDelim: "}}",
		CofferMaxMemoryBytes:       32 << 20,
		CofferAssetRoot:            "assets",
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

	a.gzipWriterPool = &sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, a.GzipCompressionLevel)
			return w
		},
	}

	a.reverseProxyTransport = newReverseProxyTransport()
	a.reverseProxyBufferPool = newReverseProxyBufferPool()

	return a
}

// GET registers a new GET route for the path with the matching h in the router
// of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodGet, path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The way, the gases is always FILO.
func (a *Air) HEAD(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodHead, path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPost, path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPut, path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) PATCH(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPatch, path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodDelete, path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) CONNECT(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodConnect, path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) OPTIONS(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodOptions, path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
func (a *Air) TRACE(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodTrace, path, h, gases...)
}

// BATCH registers a batch of routes for the methods and the path with the
// matching h in the router of the a with the optional route-level gases.
//
// The methods must either be nil (means all) or consists of one or more of the
// "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS" and
// "TRACE". Invalid methods will be silently ignored.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
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

// FILE registers a new GET route and a new HEAD route with the path in the
// router of the a to serve a static file with the filename and the optional
// route-level gases.
//
// The path may consist of static component(s), param component(s) and any param
// component.
//
// The gases is always FILO.
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

// FILES registers a new GET route and a new HEAD route with the path prefix in
// the router of the a to serve the static files from the root with the optional
// route-level gases.
//
// The path prefix may consits of static component(s) and param component(s).
// But it must not contain an any param component.
//
// The gases is always FILO.
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
		path := req.Param("*").Value().String()
		path = filepath.FromSlash(fmt.Sprint("/", path))
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
// optional group-level gases that inherited from the a.
//
// The path prefix may consits of static component(s) and param component(s).
// But it must not contain an any param component.
//
// The gases is always FILO.
func (a *Air) Group(prefix string, gases ...Gas) *Group {
	return &Group{
		Air:    a,
		Prefix: prefix,
		Gases:  gases,
	}
}

// Serve starts the server of the a.
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
		case ".toml":
			err = toml.Unmarshal(b, &m)
		case ".yaml", ".yml":
			err = yaml.Unmarshal(b, &m)
		default:
			err = fmt.Errorf(
				"air: unsupported configuration file "+
					"extension: %s",
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

// Close closes the server of the a immediately.
func (a *Air) Close() error {
	return a.server.close()
}

// Shutdown gracefully shuts down the server of the a without interrupting any
// active connections. It works by first closing all open listeners, then
// closing all idle connections, and then waiting indefinitely for connections
// to return to idle and then shut down. If the ctx expires before the shutdown
// is complete, it returns the context's error, otherwise it returns any error
// returned from closing the underlying listener(s) of the server of the a.
//
// When the `Shutdown` is called, the `Serve` immediately return the
// `http.ErrServerClosed`. Make sure the program does not exit and waits instead
// for the `Shutdown` to return.
//
// The `Shutdown` does not attempt to close nor wait for hijacked connections
// such as WebSockets. The caller should separately notify such long-lived
// connections of shutdown and wait for them to close, if desired.
func (a *Air) Shutdown(ctx context.Context) error {
	return a.server.shutdown(ctx)
}

// Addresses returns all TCP addresses that the server of the a actually listens
// on.
func (a *Air) Addresses() []string {
	return a.server.addresses()
}

// logErrorf logs the v as an error in the format.
func (a *Air) logErrorf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if a.ErrorLogger != nil {
		a.ErrorLogger.Output(2, s)
	} else {
		log.Output(2, s)
	}
}

// Handler defines a function to serve requests.
type Handler func(*Request, *Response) error

// WrapHTTPHandler provides a convenient way to wrap an `http.Handler` into a
// `Handler`.
func WrapHTTPHandler(hh http.Handler) Handler {
	return func(req *Request, res *Response) error {
		hh.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}
}

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
	if res.Written {
		return
	}

	if !req.Air.DebugMode && res.Status == http.StatusInternalServerError {
		res.WriteString(http.StatusText(res.Status))
	} else {
		res.WriteString(err.Error())
	}
}

// Gas defines a function to process gases.
//
// A gas is a function chained in the request-response cycle with access to the
// `Request` and the `Response` which it uses to perform a specific action, for
// example, logging every request or recovering from panics.
//
// The argument is the next `Handler` that the gas will be called.
//
// The return value is the gas that is wrapped into a `Handler`.
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

// splitPathQuery splits the p of the form "path?query" into path and query.
func splitPathQuery(p string) (path, query string) {
	i, l := 0, len(p)
	for ; i < l && p[i] != '?'; i++ {
	}

	if i < l {
		return p[:i], p[i+1:]
	}

	return p, ""
}

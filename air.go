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
"posts" and "assets" are STATIC components, ":UserID" and ":PostID" are PARAM
components, "*" is an ANY component. Note that all route params (PARAM and ANY
components) will be parsed into the `Request` and can be accessed via the
`Request.Param` and `Request.Params`. The name of a `RequestParam` parsed from a
PARAM component always discards its leading ":", such as ":UserID" will become
"UserID". The name of a `RequestParam` parsed from an ANY component is "*".

The second param is a `Handler` that serves the requests that match this route.
*/
package air

import (
	"compress/gzip"
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"
)

// Air is the top-level struct of this framework.
//
// It is highly recommended not to modify the value of any field of the `Air`
// after calling the `Air.Serve`, which will cause unpredictable problems.
//
// The new instances of the `Air` should only be created by calling the `New`.
// If you only need one instance of the `Air`, it is recommended to use the
// `Default`, which will help you simplify the scope management.
type Air struct {
	// AppName is the name of the web application.
	//
	// It is recommended to set the `AppName` and try to ensure that it is
	// unique (used to distinguish between different web applications).
	//
	// Default value: "air"
	AppName string `mapstructure:"app_name"`

	// MaintainerEmail is the e-mail address of the one who is responsible
	// for maintaining the web application.
	//
	// It is recommended to set the `MaintainerEmail` if the `ACMEEnabled`
	// is true (used by the CAs, such as Let's Encrypt, to notify about
	// problems with issued certificates).
	//
	// Default value: ""
	MaintainerEmail string `mapstructure:"maintainer_email"`

	// DebugMode indicates whether the web application is in debug mode.
	//
	// Please keep in mind that the `DebugMode` is quite bossy, some
	// features will be affected if it is true. So never set the `DebugMode`
	// to true in a production environment unless you want to do something
	// crazy.
	//
	// Default value: false
	DebugMode bool `mapstructure:"debug_mode"`

	// Address is the TCP address that the server listens on.
	//
	// The `Address` is never empty and contains a free port. If the port of
	// the `Address` is "0", a random port is automatically chosen. The
	// `Addresses` can be used to discover the chosen port.
	//
	// Default value: "localhost:8080"
	Address string `mapstructure:"address"`

	// ReadTimeout is the maximum duration allowed for the server to read a
	// request entirely, including the body part.
	//
	// The `ReadTimeout` does not let the `Handler` make per-request
	// decisions on each request body's acceptable deadline or upload rate.
	//
	// Default value: 0
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// ReadHeaderTimeout is the maximum duration allowed for the server to
	// read the headers of a request.
	//
	// The connection's read deadline is reset after reading the headers of
	// a request and the `Handler` can decide what is considered too slow
	// for the body.
	//
	// If the `ReadHeaderTimeout` is zero, the value of the `ReadTimeout` is
	// used. If both are zero, there is no timeout.
	//
	// Default value: 0
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`

	// WriteTimeout is the maximum duration allowed for the server to write
	// a response.
	//
	// The `WriteTimeout` is reset whenever the headers of a new request are
	// read. Like the `ReadTimeout`, the `WriteTimeout` does not let the
	// `Handler` make decisions on a per-request basis.
	//
	// Default value: 0
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum duration allowed for the server to wait
	// for the next request.
	//
	// If the `IdleTimeout` is zero, the value of the `ReadTimeout` is used.
	// If both are zero, there is no timeout.
	//
	// Default value: 0
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// MaxHeaderBytes is the maximum number of bytes allowed for the server
	// to read parsing the request headers' names and values, including
	// HTTP/1.x request-line.
	//
	// Default value: 1048576
	MaxHeaderBytes int `mapstructure:"max_header_bytes"`

	// TLSConfig is the TLS configuration to make the server to handle
	// requests on incoming TLS connections.
	//
	// Default value: nil
	TLSConfig *tls.Config `mapstructure:"-"`

	// TLSCertFile is the path to the TLS certificate file.
	//
	// The `TLSCertFile` must be set together wth the `TLSKeyFile`.
	//
	// If the certificate targeted by the `TLSCertFile` is signed by a CA,
	// it should be the concatenation of the certificate, any intermediates,
	// and the CA's certificate.
	//
	// If the `TLSConfig` is not nil, the certificate targeted by the
	// `TLSCertFile` will be appended to the end of the `Certificates` of
	// the `TLSConfig`'s clone. Otherwise, a new instance of the
	// `tls.Config` will be created with the certificate.
	//
	// Default value: ""
	TLSCertFile string `mapstructure:"tls_cert_file"`

	// TLSKeyFile is the path to the TLS key file.
	//
	// The key targeted by the `TLSKeyFile` must match the certificate
	// targeted by the `TLSCertFile`.
	//
	// Default value: ""
	TLSKeyFile string `mapstructure:"tls_key_file"`

	// ACMEEnabled indicates whether the ACME feature is enabled.
	//
	// The `ACMEEnabled` gives the server the ability to automatically
	// obtain new certificates from the ACME CA.
	//
	// If the `TLSConfig` and `TLSConfig.GetCertificate` are not nil, the
	// server will respect it and use the ACME feature as a backup.
	// Otherwise, a new instance of the `tls.Config` will be created with
	// the ACME feature.
	//
	// Default value: false
	ACMEEnabled bool `mapstructure:"acme_enabled"`

	// ACMEDirectoryURL is the ACME CA directory URL of the ACME feature.
	//
	// Default value: "https://acme-v02.api.letsencrypt.org/directory"
	ACMEDirectoryURL string `mapstructure:"acme_directory_url"`

	// ACMETOSURLWhitelist is the list of ACME CA's Terms of Service (TOS)
	// URL allowed by the ACME feature.
	//
	// If the length of the `ACMETOSURLWhitelist` is zero, all TOS URLs will
	// be allowed.
	//
	// Default value: nil
	ACMETOSURLWhitelist []string `mapstructure:"acme_tos_url_whitelist"`

	// ACMEAccountKey is the account key of the ACME feature used to
	// register with an ACME CA and sign requests.
	//
	// Supported algorithms:
	//   * RS256
	//   * ES256
	//   * ES384
	//   * ES512
	//
	// If the `ACMEAccountKey` is nil, a new ECDSA P-256 key is generated.
	//
	// Default value: nil
	ACMEAccountKey crypto.Signer `mapstructure:"-"`

	// ACMECertRoot is the root of the certificates of the ACME feature.
	//
	// It is recommended to set the `ACMECertRoot` since all ACME CAs have a
	// rate limit on issuing certificates. Different web applications can
	// share the same place (if they are all built using this framework).
	//
	// Default value: "acme-certs"
	ACMECertRoot string `mapstructure:"acme_cert_root"`

	// ACMEDefaultHost is the default host of the ACME feature.
	//
	// The `ACMEDefaultHost` is only used when the host is missing from the
	// TLS handshake.
	//
	// Default value: ""
	ACMEDefaultHost string `mapstructure:"acme_default_host"`

	// ACMEHostWhitelist is the list of hosts allowed by the ACME feature.
	//
	// It is highly recommended to set the `ACMEHostWhitelist`. If the
	// length of the `ACMEHostWhitelist` is not zero, all connections that
	// are not connected to the hosts in it will not be able to obtain new
	// certificates from the ACME CA.
	//
	// Default value: nil
	ACMEHostWhitelist []string `mapstructure:"acme_host_whitelist"`

	// ACMERenewalWindow is the renewal window of the ACME feature before a
	// certificate expires.
	//
	// Default value: 2592000000000000
	ACMERenewalWindow time.Duration `mapstructure:"acme_renewal_window"`

	// ACMEExtraExts is the list of extra extensions used when generating a
	// new CSR (Certificate Request), thus allowing customization of the
	// resulting certificate.
	//
	// Default value: nil
	ACMEExtraExts []pkix.Extension `mapstructure:"-"`

	// HTTPSEnforced indicates whether the server is forcibly accessible
	// only via the HTTPS scheme (HTTP requests will be automatically
	// redirected to HTTPS).
	//
	// The `HTTPSEnforced` will always be treated as true when the
	// `ACMEEnabled` is true.
	//
	// Default value: false
	HTTPSEnforced bool `mapstructure:"https_enforced"`

	// HTTPSEnforcedPort is the port of the TCP address (share the same host
	// as the `Address`) that the server listens on. All requests to this
	// port will be automatically redirected to HTTPS.
	//
	// If the `HTTPSEnforcedPort` is "0", a random port is automatically
	// chosen. The `Addresses` can be used to discover the chosen port.
	//
	// Default value: "0"
	HTTPSEnforcedPort string `mapstructure:"https_enforced_port"`

	// WebSocketHandshakeTimeout is the maximum duration allowed for the
	// server to wait for a WebSocket handshake to complete.
	//
	// Default value: 0
	WebSocketHandshakeTimeout time.Duration `mapstructure:"websocket_handshake_timeout"`

	// WebSocketSubprotocols is the list of supported WebSocket subprotocols
	// of the server.
	//
	// If the length of the `WebSocketSubprotocols` is not zero, the
	// `Response.WebSocket` negotiates a subprotocol by selecting the first
	// match with a protocol requested by the client. If there is no match,
	// no protocol is negotiated (the Sec-Websocket-Protocol header is not
	// included in the handshake response).
	//
	// Default value: nil
	WebSocketSubprotocols []string `mapstructure:"websocket_subprotocols"`

	// PROXYEnabled indicates whether the PROXY feature is enabled.
	//
	// The `PROXYEnabled` gives the server the ability to support the PROXY
	// protocol (See
	// https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt).
	//
	// Default value: false
	PROXYEnabled bool `mapstructure:"proxy_enabled"`

	// PROXYReadHeaderTimeout is the maximum duration allowed for the server
	// to read the PROXY protocol header of a connection.
	//
	// The connection's read deadline is reset after reading the PROXY
	// protocol header.
	//
	// Default value: 0
	PROXYReadHeaderTimeout time.Duration `mapstructure:"proxy_read_header_timeout"`

	// PROXYRelayerIPWhitelist is the list of IP addresses or CIDR notation
	// IP address ranges of the relayers allowed by the PROXY feature.
	//
	// It is highly recommended to set the `PROXYRelayerIPWhitelist`. If the
	// length of the `PROXYRelayerIPWhitelist` is not zero, all connections
	// relayed from the IP addresses are not in it will not be able to act
	// the PROXY protocol.
	//
	// Default value: nil
	PROXYRelayerIPWhitelist []string `mapstructure:"proxy_relayer_ip_whitelist"`

	// Pregases is the `Gas` chain stack that performs before routing.
	//
	// The `Pregases` is always FILO.
	//
	// Default value: nil
	Pregases []Gas `mapstructure:"-"`

	// Gases is the `Gas` chain stack that performs after routing.
	//
	// The `Gases` is always FILO.
	//
	// Default value: nil
	Gases []Gas `mapstructure:"-"`

	// NotFoundHandler is the `Handler` that returns not found error.
	//
	// The `NotFoundHandler` is never nil because the router will use it as
	// the default `Handler` when no match is found.
	//
	// Default value: `DefaultNotFoundHandler`
	NotFoundHandler func(*Request, *Response) error `mapstructure:"-"`

	// MethodNotAllowedHandler is the `Handler` that returns method not
	// allowed error.
	//
	// The `MethodNotAllowedHandler` is never nil because the router will
	// use it as the default `Handler` when a match is found but the request
	// method is not registered.
	//
	// Default value: `DefaultMethodNotAllowedHandler`
	MethodNotAllowedHandler func(*Request, *Response) error `mapstructure:"-"`

	// ErrorHandler is the centralized error handler.
	//
	// The `ErrorHandler` is never nil because the server will use it in
	// every request-response cycle that has an error.
	//
	// Default value: `DefaultErrorHandler`
	ErrorHandler func(error, *Request, *Response) `mapstructure:"-"`

	// ErrorLogger is the `log.Logger` that logs errors that occur in the
	// web application.
	//
	// If the `ErrorLogger` is nil, logging is done via the log package's
	// standard logger.
	//
	// Default value: nil
	ErrorLogger *log.Logger `mapstructure:"-"`

	// RendererTemplateRoot is the root of the HTML templates of the
	// renderer feature.
	//
	// All HTML template files inside the `RendererTemplateRoot` will be
	// recursively parsed into the renderer and their names will be used as
	// HTML template names.
	//
	// Default value: "templates"
	RendererTemplateRoot string `mapstructure:"renderer_template_root"`

	// RendererTemplateExts is the list of filename extensions of the HTML
	// templates of the renderer feature used to distinguish the HTML
	// template files in the `RendererTemplateRoot`.
	//
	// Default value: [".html"]
	RendererTemplateExts []string `mapstructure:"renderer_template_exts"`

	// RendererTemplateLeftDelim is the left side of the HTML template
	// delimiter of the renderer feature.
	//
	// default value: "{{"
	RendererTemplateLeftDelim string `mapstructure:"renderer_template_left_delim"`

	// RendererTemplateRightDelim is the right side of the HTML template
	// delimiter of the renderer feature.
	//
	// Default value: "}}"
	RendererTemplateRightDelim string `mapstructure:"renderer_template_right_delim"`

	// RendererTemplateFuncMap is the HTML template function map of the
	// renderer feature.
	//
	// Default value: nil
	RendererTemplateFuncMap template.FuncMap `mapstructure:"-"`

	// MinifierEnabled indicates whether the minifier feature is enabled.
	//
	// The `MinifierEnabled` gives the `Response.Write` the ability to
	// minify the matching response body on the fly based on the
	// Content-Type header.
	//
	// Default value: false
	MinifierEnabled bool `mapstructure:"minifier_enabled"`

	// MinifierMIMETypes is the list of MIME types of the minifier feature
	// that will trigger the minimization.
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

	// GzipEnabled indicates whether the gzip feature is enabled.
	//
	// The `GzipEnabled` gives the `Response` the ability to gzip the
	// matching response body on the fly based on the Content-Type header.
	//
	// Default value: false
	GzipEnabled bool `mapstructure:"gzip_enabled"`

	// GzipMIMETypes is the list of MIME types of the gzip feature that will
	// trigger the gzip.
	//
	// Default value: ["text/plain", "text/html", "text/css",
	// "application/javascript", "application/json", "application/xml",
	// "application/toml", "application/yaml", "image/svg+xml"]
	GzipMIMETypes []string `mapstructure:"gzip_mime_types"`

	// GzipCompressionLevel is the compression level of the gzip feature.
	//
	// Default value: `gzip.DefaultCompression`
	GzipCompressionLevel int `mapstructure:"gzip_compression_level"`

	// GzipMinContentLength is the minimum content length of the gzip
	// featrue used to limit at least how big (determined only from the
	// Content-Length header) response body can be gzipped.
	//
	// Default value: 1024
	GzipMinContentLength int64 `mapstructure:"gzip_min_content_length"`

	// CofferEnabled indicates whether the coffer feature is enabled.
	//
	// The `CofferEnabled` gives the `Response.WriteFile` the ability to use
	// the runtime memory to reduce the disk I/O pressure.
	//
	// Default value: false
	CofferEnabled bool `mapstructure:"coffer_enabled"`

	// CofferMaxMemoryBytes is the maximum number of bytes of the runtime
	// memory allowed for the coffer feature to use.
	//
	// Default value: 33554432
	CofferMaxMemoryBytes int `mapstructure:"coffer_max_memory_bytes"`

	// CofferAssetRoot is the root of the assets of the coffer feature.
	//
	// All asset files inside the `CofferAssetRoot` will be recursively
	// parsed into the coffer and their names will be used as asset names.
	//
	// Default value: "assets"
	CofferAssetRoot string `mapstructure:"coffer_asset_root"`

	// CofferAssetExts is the list of filename extensions of the assets of
	// the coffer feature used to distinguish the asset files in the
	// `CofferAssetRoot`.
	//
	// Default value: [".html", ".css", ".js", ".json", ".xml", ".toml",
	// ".yaml", ".yml", ".svg", ".jpg", ".jpeg", ".png", ".gif"]
	CofferAssetExts []string `mapstructure:"coffer_asset_exts"`

	// I18nEnabled indicates whether the i18n feature is enabled.
	//
	// The `I18nEnabled` gives the `Request.LocalizedString` and
	// `Response.Render` the ability to adapt to the request's favorite
	// conventions based on the Accept-Language header.
	//
	// Default value: false
	I18nEnabled bool `mapstructure:"i18n_enabled"`

	// I18nLocaleRoot is the root of the locales of the i18n feature.
	//
	// All TOML-based locale files (".toml" is the extension) inside the
	// `I18nLocaleRoot` will be parsed into the i18n and their names
	// (without extension) will be used as locales.
	//
	// Default value: "locales"
	I18nLocaleRoot string `mapstructure:"i18n_locale_root"`

	// I18nLocaleBase is the base of the locales of the i18n feature used
	// when a locale cannot be found.
	//
	// Default value: "en-US"
	I18nLocaleBase string `mapstructure:"i18n_locale_base"`

	// ConfigFile is the path to the configuration file that will be parsed
	// into the matching fields before starting the server.
	//
	// The ".json" extension means the configuration file is JSON-based.
	//
	// The ".toml" extension means the configuration file is TOML-based.
	//
	// The ".yaml" and ".yml" extensions means the configuration file is
	// YAML-based.
	//
	// Default value: ""
	ConfigFile string `mapstructure:"-"`

	server   *http.Server
	router   *router
	binder   *binder
	renderer *renderer
	minifier *minifier
	coffer   *coffer
	i18n     *i18n

	addressMap                   map[string]int
	shutdownJobs                 []func()
	shutdownJobMutex             *sync.Mutex
	shutdownJobDone              chan struct{}
	requestPool                  *sync.Pool
	responsePool                 *sync.Pool
	contentTypeSnifferBufferPool *sync.Pool
	gzipWriterPool               *sync.Pool
	reverseProxyTransport        *reverseProxyTransport
	reverseProxyBufferPool       *reverseProxyBufferPool
}

// Default is the default instance of the `Air`.
//
// If you only need one instance of the `Air`, you should use the `Default`.
// Unless you think you can efficiently pass your instance in different scopes.
var Default = New()

// New returns a new instance of the `Air` with default field values.
//
// The `New` is the only function that creates new instances of the `Air` and
// keeps everything working.
func New() *Air {
	a := &Air{
		AppName:                 "air",
		Address:                 "localhost:8080",
		MaxHeaderBytes:          1 << 20,
		ACMEDirectoryURL:        "https://acme-v02.api.letsencrypt.org/directory",
		ACMECertRoot:            "acme-certs",
		ACMERenewalWindow:       30 * 24 * time.Hour,
		HTTPSEnforcedPort:       "0",
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
		GzipMinContentLength:       1 << 10,
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

	a.server = &http.Server{}
	a.router = newRouter(a)
	a.binder = newBinder(a)
	a.renderer = newRenderer(a)
	a.minifier = newMinifier(a)
	a.coffer = newCoffer(a)
	a.i18n = newI18n(a)

	a.addressMap = map[string]int{}
	a.shutdownJobMutex = &sync.Mutex{}
	a.shutdownJobDone = make(chan struct{})
	a.requestPool = &sync.Pool{
		New: func() interface{} {
			return &Request{}
		},
	}

	a.responsePool = &sync.Pool{
		New: func() interface{} {
			return &Response{}
		},
	}

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
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) GET(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodGet, path, h, gases...)
}

// HEAD registers a new HEAD route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) HEAD(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodHead, path, h, gases...)
}

// POST registers a new POST route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) POST(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPost, path, h, gases...)
}

// PUT registers a new PUT route for the path with the matching h in the router
// of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) PUT(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPut, path, h, gases...)
}

// PATCH registers a new PATCH route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) PATCH(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodPatch, path, h, gases...)
}

// DELETE registers a new DELETE route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) DELETE(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodDelete, path, h, gases...)
}

// CONNECT registers a new CONNECT route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) CONNECT(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodConnect, path, h, gases...)
}

// OPTIONS registers a new OPTIONS route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) OPTIONS(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodOptions, path, h, gases...)
}

// TRACE registers a new TRACE route for the path with the matching h in the
// router of the a with the optional route-level gases.
//
// The path may consist of STATIC, PARAM and ANY components.
//
// The gases is always FILO.
func (a *Air) TRACE(path string, h Handler, gases ...Gas) {
	a.router.register(http.MethodTrace, path, h, gases...)
}

// BATCH registers a batch of routes for the methods and path with the matching
// h in the router of the a with the optional route-level gases.
//
// The methods must either be nil (means all) or consists of one or more of the
// "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS" and
// "TRACE". Invalid methods will be silently ignored.
//
// The path may consist of STATIC, PARAM and ANY components.
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

// FILE registers a new GET and HEAD route pair with the path in the router of
// the a to serve a static file with the filename and optional route-level
// gases.
//
// The path may consist of STATIC, PARAM and ANY components.
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

// FILES registers some new GET and HEAD route paris with the path prefix in the
// router of the a to serve the static files from the root with the optional
// route-level gases.
//
// The prefix may consit of STATIC and PARAM components, but it must not contain
// ANY component.
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

// Group returns a new instance of the `Group` with the path prefix and optional
// group-level gases that inherited from the a.
//
// The prefix may consit of STATIC and PARAM components, but it must not contain
// ANY component.
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

	host, port, err := net.SplitHostPort(a.Address)
	if err != nil {
		return err
	}

	a.server.Addr = net.JoinHostPort(host, port)
	a.server.Handler = a
	a.server.ReadTimeout = a.ReadTimeout
	a.server.ReadHeaderTimeout = a.ReadHeaderTimeout
	a.server.WriteTimeout = a.WriteTimeout
	a.server.IdleTimeout = a.IdleTimeout
	a.server.MaxHeaderBytes = a.MaxHeaderBytes
	a.server.ErrorLog = a.ErrorLogger

	tlsConfig := a.TLSConfig
	if tlsConfig != nil {
		tlsConfig = tlsConfig.Clone()
	}

	if a.TLSCertFile != "" && a.TLSKeyFile != "" {
		c, err := tls.LoadX509KeyPair(a.TLSCertFile, a.TLSKeyFile)
		if err != nil {
			return err
		}

		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, c)
	}

	if tlsConfig != nil {
		for _, proto := range []string{"h2", "http/1.1"} {
			if !stringSliceContains(
				tlsConfig.NextProtos,
				proto,
				false,
			) {
				tlsConfig.NextProtos = append(
					tlsConfig.NextProtos,
					proto,
				)
			}
		}
	}

	hh := http.Handler(http.HandlerFunc(func(
		rw http.ResponseWriter,
		r *http.Request,
	) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}

		if port != "443" {
			host = net.JoinHostPort(host, port)
		}

		http.Redirect(
			rw,
			r,
			fmt.Sprint("https://", host, r.RequestURI),
			http.StatusMovedPermanently,
		)
	}))

	if a.ACMEEnabled {
		acm := &autocert.Manager{
			Prompt: func(tosURL string) bool {
				if len(a.ACMETOSURLWhitelist) == 0 {
					return true
				}

				for _, u := range a.ACMETOSURLWhitelist {
					if u == tosURL {
						return true
					}
				}

				return false
			},
			Cache:       autocert.DirCache(a.ACMECertRoot),
			RenewBefore: a.ACMERenewalWindow,
			Client: &acme.Client{
				Key:          a.ACMEAccountKey,
				DirectoryURL: a.ACMEDirectoryURL,
			},
			Email:           a.MaintainerEmail,
			ExtraExtensions: a.ACMEExtraExts,
		}
		if a.ACMEHostWhitelist != nil {
			acm.HostPolicy = autocert.HostWhitelist(
				a.ACMEHostWhitelist...,
			)
		}

		hh = acm.HTTPHandler(hh)

		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}

		getCertificate := tlsConfig.GetCertificate
		tlsConfig.GetCertificate = func(
			chi *tls.ClientHelloInfo,
		) (*tls.Certificate, error) {
			if getCertificate != nil {
				c, err := getCertificate(chi)
				if err != nil {
					return nil, err
				}

				if c != nil {
					return c, nil
				}
			}

			if chi.ServerName == "" {
				chi.ServerName = a.ACMEDefaultHost
			}

			return acm.GetCertificate(chi)
		}

		for _, proto := range acm.TLSConfig().NextProtos {
			if !stringSliceContains(
				tlsConfig.NextProtos,
				proto,
				false,
			) {
				tlsConfig.NextProtos = append(
					tlsConfig.NextProtos,
					proto,
				)
			}
		}
	}

	listener := newListener(a)
	if err := listener.listen(a.server.Addr); err != nil {
		return err
	}
	defer listener.Close()

	a.addressMap[listener.Addr().String()] = 0
	defer delete(a.addressMap, listener.Addr().String())

	netListener := net.Listener(listener)
	httpsEnforced := a.HTTPSEnforced || a.ACMEEnabled
	if tlsConfig != nil {
		netListener = tls.NewListener(netListener, tlsConfig)
		if httpsEnforced {
			hs := &http.Server{
				Addr: net.JoinHostPort(
					host,
					a.HTTPSEnforcedPort,
				),
				Handler:           hh,
				ReadTimeout:       a.ReadTimeout,
				ReadHeaderTimeout: a.ReadHeaderTimeout,
				WriteTimeout:      a.WriteTimeout,
				IdleTimeout:       a.IdleTimeout,
				MaxHeaderBytes:    a.MaxHeaderBytes,
				ErrorLog:          a.ErrorLogger,
			}

			l := newListener(a)
			if err := l.listen(hs.Addr); err != nil {
				return err
			}
			defer l.Close()

			a.addressMap[l.Addr().String()] = 1
			defer delete(a.addressMap, l.Addr().String())

			go hs.Serve(l)
			defer hs.Close()
		}
	} else {
		h2s := &http2.Server{
			IdleTimeout: a.IdleTimeout,
		}
		if h2s.IdleTimeout == 0 {
			h2s.IdleTimeout = a.ReadTimeout
		}

		a.server.Handler = h2c.NewHandler(a.server.Handler, h2s)
	}

	if port == "0" || (httpsEnforced && a.HTTPSEnforcedPort == "0") {
		_, port, _ = net.SplitHostPort(netListener.Addr().String())
		fmt.Printf("air: listening on %v\n", a.Addresses())
	}

	shutdownJobRunOnce := sync.Once{}
	a.server.RegisterOnShutdown(func() {
		a.shutdownJobMutex.Lock()
		defer a.shutdownJobMutex.Unlock()
		shutdownJobRunOnce.Do(func() {
			waitGroup := sync.WaitGroup{}
			for _, job := range a.shutdownJobs {
				if job != nil {
					waitGroup.Add(1)
					go func(job func()) {
						job()
						waitGroup.Done()
					}(job)
				}
			}

			waitGroup.Wait()

			close(a.shutdownJobDone)
		})
	})

	if a.DebugMode {
		fmt.Println("air: serving in debug mode")
	}

	return a.server.Serve(netListener)
}

// Close closes the server of the a immediately.
func (a *Air) Close() error {
	return a.server.Close()
}

// Shutdown gracefully shuts down the server of the a without interrupting any
// active connections. It works by first closing all open listeners, then start
// running all shutdown jobs added via the `AddShutdownJob` concurrently, and
// then closing all idle connections, and then waiting indefinitely for
// connections to return to idle and shutdown jobs to complete and then shut
// down. If the ctx expires before the shutdown is complete, it returns the
// context's error, otherwise it returns any error returned from closing the
// underlying listener(s) of the server of the a.
//
// When the `Shutdown` is called, the `Serve` immediately return the
// `http.ErrServerClosed`. Make sure the program does not exit and waits instead
// for the `Shutdown` to return.
//
// The `Shutdown` does not attempt to close nor wait for hijacked connections
// such as WebSockets. The caller should separately notify such long-lived
// connections of shutdown and wait for them to close, if desired. See the
// `AddShutdownJob` for a way to add shutdown jobs.
func (a *Air) Shutdown(ctx context.Context) error {
	err := a.server.Shutdown(ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.shutdownJobDone:
	}

	return err
}

// AddShutdownJob adds the f as a shutdown job that will run only once when the
// `Shutdown` is called. The return value is an unique ID assigned to the f,
// which can be used to remove the f from the shutdown job queue by calling the
// `RemoveShutdownJob`.
func (a *Air) AddShutdownJob(f func()) int {
	a.shutdownJobMutex.Lock()
	defer a.shutdownJobMutex.Unlock()
	a.shutdownJobs = append(a.shutdownJobs, f)
	return len(a.shutdownJobs) - 1
}

// RemoveShutdownJob removes the shutdown job targeted by the id from the
// shutdown job queue.
func (a *Air) RemoveShutdownJob(id int) {
	a.shutdownJobMutex.Lock()
	defer a.shutdownJobMutex.Unlock()
	if id >= 0 && id < len(a.shutdownJobs) {
		a.shutdownJobs[id] = nil
	}
}

// Addresses returns all TCP addresses that the server of the a actually listens
// on.
func (a *Air) Addresses() []string {
	asl := len(a.addressMap)
	if asl == 0 {
		return nil
	}

	as := make([]string, 0, asl)
	for a := range a.addressMap {
		as = append(as, a)
	}

	sort.Slice(as, func(i, j int) bool {
		return a.addressMap[as[i]] < a.addressMap[as[j]]
	})

	return as
}

// ServeHTTP implements the `http.Handler`.
func (a *Air) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Get the request and response from the pool.

	req := a.requestPool.Get().(*Request)
	res := a.responsePool.Get().(*Response)

	req.reset(a, r, res)
	res.reset(a, rw, req)

	// Chain the gases stack.

	h := func(req *Request, res *Response) error {
		h := a.router.route(req)
		for i := len(a.Gases) - 1; i >= 0; i-- {
			h = a.Gases[i](h)
		}

		return h(req, res)
	}

	// Chain the pregases stack.

	for i := len(a.Pregases) - 1; i >= 0; i-- {
		h = a.Pregases[i](h)
	}

	// Execute the chain.

	if err := h(req, res); err != nil {
		if !res.Written && res.Status < http.StatusBadRequest {
			res.Status = http.StatusInternalServerError
		}

		a.ErrorHandler(err, req, res)
	}

	// Execute the deferred functions.

	for i := len(res.deferredFuncs) - 1; i >= 0; i-- {
		res.deferredFuncs[i]()
	}

	// Put the route param values back to the pool.

	if req.routeParamValues != nil {
		a.router.routeParamValuesPool.Put(req.routeParamValues)
	}

	// Put the request and response back to the pool.

	a.requestPool.Put(req)
	a.responsePool.Put(res)
}

// logErrorf logs the v as an error in the format.
func (a *Air) logErrorf(format string, v ...interface{}) {
	e := fmt.Errorf(format, v...)
	if a.ErrorLogger != nil {
		a.ErrorLogger.Output(2, e.Error())
	} else {
		log.Output(2, e.Error())
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

// DefaultErrorHandler is the default centralized error handler.
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
// `Request` and `Response` which it uses to perform a specific action, for
// example, logging every request or recovering from panics.
//
// The param is the next `Handler` that the gas will be called.
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

// stringSliceContains reports whether the ss contains the s. The
// caseInsensitive indicates whether to ignore case when comparing.
func stringSliceContains(ss []string, s string, caseInsensitive bool) bool {
	if caseInsensitive {
		for _, v := range ss {
			if strings.EqualFold(v, s) {
				return true
			}
		}

		return false
	}

	for _, v := range ss {
		if v == s {
			return true
		}
	}

	return false
}

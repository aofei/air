package air

import (
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is a global set of configs that for an instance of the `Air` for
// customization.
type Config struct {
	// AppName represents the name of the `Air` instance.
	//
	// The default Value is "air".
	//
	// It's called "app_name" in the config file.
	AppName string

	// DebugMode indicates whether to enable the debug mode when the HTTP
	// server is started.
	//
	// The default value is false.
	//
	// It's called "debug_mode" in the config file.
	DebugMode bool

	// LoggerEnabled indicates whether to enable the `Logger` when the HTTP
	// server is started. It works only with the default `Logger`.
	//
	// It will be forced to the true if the `DebugMode` is true.
	//
	// The default value is false.
	//
	// It's called "logger_enabled" in the config file.
	LoggerEnabled bool

	// LogFormat represents the format of the output content of the
	// `Logger`. It works only with the default `Logger` and when the
	// `LoggerEnabled` is true.
	//
	// The default value is:
	// `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}",` +
	// `"level":"{{.level}}","file":"{{.short_file}}","line":"{{.line}}"}`
	//
	// It's called "log_format" in the config file.
	LogFormat string

	// Address represents the TCP address that the HTTP server to listen on.
	//
	// The default value is "localhost:2333".
	//
	// It's called "address" in the config file.
	Address string

	// ReadTimeout represents the maximum duration before timing out read of
	// the HTTP request.
	//
	// The default value is 0.
	//
	// It's called "read_timeout" in the config file.
	//
	// **It's unit in the config file is MILLISECONDS.**
	ReadTimeout time.Duration

	// WriteTimeout represents the maximum duration before timing out write
	// of the HTTP response.
	//
	// The default value is 0.
	//
	// It's called "write_timeout" in the config file.
	//
	// **It's unit in the config file is MILLISECONDS.**
	WriteTimeout time.Duration

	// MaxHeaderBytes represents the maximum number of bytes the HTTP server
	// will read parsing the HTTP request header's keys and values,
	// including the HTTP request line. It does not limit the size of the
	// HTTP request body.
	//
	// The default value is 1048576.
	//
	// It's called "max_header_bytes" in the config file.
	MaxHeaderBytes int

	// TLSCertFile represents the path of the TLS certificate file.
	//
	// The default value is "".
	//
	// It's called "tls_cert_file" in the config file.
	TLSCertFile string

	// TLSKeyFile represents the path of the TLS key file.
	//
	// The default value is "".
	//
	// It's called "tls_key_file" in the config file.
	TLSKeyFile string

	// MinifierEnabled indicates whether to enable the `Minifier` when the
	// HTTP server is started. It works only with the default `Minifier`.
	//
	// The default value is false.
	//
	// It's called "minifier_enabled" in the config file.
	MinifierEnabled bool

	// TemplateRoot represents the root directory of the HTML templates. It
	// will be parsed into the `Renderer`. It works only with the default
	// `Renderer`.
	//
	// The default value is "templates" that means a subdirectory of the
	// runtime directory.
	//
	// It's called "template_root" in the config file.
	TemplateRoot string

	// TemplateExts represents the file name extensions of the HTML
	// templates. It will be used when parsing the HTML templates. It works
	// only with the default `Renderer`.
	//
	// The default value is [".html"].
	//
	// It's called "template_exts" in the config file.
	TemplateExts []string

	// TemplateLeftDelim represents the left side of the HTML template
	// delimiter. It will be used when parsing the HTML templates. It works
	// only with the default `Renderer`.
	//
	// The default value is "{{".
	//
	// It's called "template_left_delim" in the config file.
	TemplateLeftDelim string

	// TemplateRightDelim represents the right side of the HTML template
	// delimiter. It will be used when parsing the HTML templates. It works
	// only with the default `Renderer`.
	//
	// The default value is "}}".
	//
	// It's called "template_right_delim" in the config file.
	TemplateRightDelim string

	// CofferEnabled indicates whether to enable the `Coffer` when the HTTP
	// server is started. It works only with the default `Coffer`.
	//
	// The default value is false.
	//
	// It's called "coffer_enabled" in the config file.
	CofferEnabled bool

	// AssetRoot represents the root directory of the asset files. It will
	// be loaded into the `Coffer`. It works only with the default `Coffer`
	// and when the `CofferEnabled` is true.
	//
	// The default value is "assets" that means a subdirectory of the
	// runtime directory.
	//
	// It's called "asset_root" in the config file.
	AssetRoot string

	// AssetExts represents the file name extensions of the asset files. It
	// will be used when loading the asset files. It works only with the
	// default `Coffer` and when the `CofferEnabled` is true.
	//
	// The default value is [".html", ".css", ".js", ".json", ".xml",
	// ".svg"].
	//
	// It's called "asset_exts" in the config file.
	AssetExts []string

	// Data represents the data that parsing from the config file. You can
	// use it to access the values in the config file.
	//
	// e.g. Data["foobar"] will accesses the value in the config file called
	// "foobar".
	Data Map
}

// DefaultConfig is the default instance of the `Config`.
var DefaultConfig = Config{
	AppName: "air",
	LogFormat: `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}",` +
		`"level":"{{.level}}","file":"{{.short_file}}",` +
		`"line":"{{.line}}"}`,
	Address:            "localhost:2333",
	MaxHeaderBytes:     1 << 20,
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
	},
}

// NewConfig returns a pointer of a new instance of the `Config` by parsing the
// config file found in the filename path. It returns a copy of the
// DefaultConfig if the config file does not exist.
func NewConfig(filename string) *Config {
	c := DefaultConfig
	c.ParseFile(filename)
	return &c
}

// Parse parses the src into the c.
func (c *Config) Parse(src string) error {
	if err := toml.Unmarshal([]byte(src), &c.Data); err != nil {
		return err
	}
	c.fillData()
	return nil
}

// ParseFile parses the config file found in the filename path into the c.
func (c *Config) ParseFile(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return c.Parse(string(b))
}

// fillData fills the values of the fields from the field `Data` of the c.
func (c *Config) fillData() {
	if an, ok := c.Data["app_name"].(string); ok {
		c.AppName = an
	}
	if dm, ok := c.Data["debug_mode"].(bool); ok {
		c.DebugMode = dm
	}
	if le, ok := c.Data["logger_enabled"].(bool); ok {
		c.LoggerEnabled = le
	}
	if lf, ok := c.Data["log_format"].(string); ok {
		c.LogFormat = lf
	}
	if a, ok := c.Data["address"].(string); ok {
		c.Address = a
	}
	if rt, ok := c.Data["read_timeout"].(int64); ok {
		c.ReadTimeout = time.Duration(rt) * time.Millisecond
	}
	if wt, ok := c.Data["write_timeout"].(int64); ok {
		c.WriteTimeout = time.Duration(wt) * time.Millisecond
	}
	if mhb, ok := c.Data["max_header_bytes"].(int64); ok {
		c.MaxHeaderBytes = int(mhb)
	}
	if tcf, ok := c.Data["tls_cert_file"].(string); ok {
		c.TLSCertFile = tcf
	}
	if tkf, ok := c.Data["tls_key_file"].(string); ok {
		c.TLSKeyFile = tkf
	}
	if me, ok := c.Data["minifier_enabled"].(bool); ok {
		c.MinifierEnabled = me
	}
	if tr, ok := c.Data["template_root"].(string); ok {
		c.TemplateRoot = tr
	}
	if tes, ok := c.Data["template_exts"].([]interface{}); ok {
		c.TemplateExts = nil
		for _, te := range tes {
			c.TemplateExts = append(c.TemplateExts, te.(string))
		}
	}
	if tld, ok := c.Data["template_left_delim"].(string); ok {
		c.TemplateLeftDelim = tld
	}
	if trd, ok := c.Data["template_right_delim"].(string); ok {
		c.TemplateRightDelim = trd
	}
	if ce, ok := c.Data["coffer_enabled"].(bool); ok {
		c.CofferEnabled = ce
	}
	if ar, ok := c.Data["asset_root"].(string); ok {
		c.AssetRoot = ar
	}
	if aes, ok := c.Data["asset_exts"].([]interface{}); ok {
		c.AssetExts = nil
		for _, ae := range aes {
			c.AssetExts = append(c.AssetExts, ae.(string))
		}
	}
}

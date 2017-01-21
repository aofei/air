package air

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config is a global set of configs that for an instance of the `Air` for customization.
type Config struct {
	// AppName represens the name of the `Air` instance.
	//
	// The default Value is "air".
	//
	// It's called "app_name" in the config file.
	AppName string

	// DebugMode represents the state of the debug mode enabled of the `Air`. It works only with
	// the default `Logger`.
	//
	// The default value is false.
	//
	// It's called "debug_mode" in the config file.
	DebugMode bool

	// LogEnabled represents the state of the enabled of the `Logger`. It will be forced to the
	// true if the `DebugMode` is true. It works only with the default `Logger`.
	//
	// The default value is false.
	//
	// It's called "log_enabled" in the config file.
	LogEnabled bool

	// LogFormat represents the format of the output content of the `Logger`. It works only with
	// the default `Logger`.
	//
	// The default value is:
	// `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}","level":"{{.level}}",` +
	// `"file":"{{.short_file}}","line":"{{.line}}"}`
	//
	// It's called "log_format" in the config file.
	LogFormat string

	// Address represents the TCP address that the HTTP server to listen on.
	//
	// The default value is "localhost:2333".
	//
	// It's called "address" in the config file.
	Address string

	// Listener represens the custom `net.Listener`. If set, the HTTP server accepts connections
	// on it.
	//
	// The default value is nil.
	Listener net.Listener

	// DisableHTTP2 represens the state of the HTTP/2 disabled of the `Air`.
	//
	// The default value is false.
	//
	// It's called "disable_http2" in the config file.
	DisableHTTP2 bool

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

	// ReadTimeout represents the maximum duration before timing out read of the HTTP request.
	//
	// The default value is 0.
	//
	// It's called "read_timeout" in the config file.
	//
	// **It's unit in the config file is SECONDS.**
	ReadTimeout time.Duration

	// WriteTimeout represents the maximum duration before timing out write of the HTTP
	// response.
	//
	// The default value is 0.
	//
	// It's called "write_timeout" in the config file.
	//
	// **It's unit in the config file is SECONDS.**
	WriteTimeout time.Duration

	// TemplateRoot represents the root directory of the HTML templates. It will be parsed into
	// the `Renderer`. It works only with the default `Renderer`.
	//
	// The default value is "templates" that means a subdirectory of the runtime directory.
	//
	// It's called "template_root" in the config file.
	TemplateRoot string

	// TemplateExt represents the file name extension of the HTML templates. It will be used
	// when parsing the HTML templates. It works only with the default `Renderer`.
	//
	// The default value is ".html".
	//
	// It's called "template_ext" in the config file.
	TemplateExt string

	// TemplateLeftDelim represents the left side of the HTML template delimiter. It will be
	// used when parsing the HTML templates. It works only with the default `Renderer`.
	//
	// The default value is "{{".
	//
	// It's called "template_left_delim" in the config file.
	TemplateLeftDelim string

	// TemplateRightDelim represents the right side of the HTML template delimiter. It will be
	// used when parsing the HTML templates. It works only with the default `Renderer`.
	//
	// The default value is "}}".
	//
	// It's called "template_right_delim" in the config file.
	TemplateRightDelim string

	// MinifyTemplate indicates whether to minify the HTML templates before they being parsed
	// into the `Renderer`. It works only with the default `Renderer`. The minify feature
	// powered by the Minify project that can be found at "https://github.com/tdewolff/minify".
	//
	// The default value is false.
	//
	// It's called "minify_template" in the config file.
	MinifyTemplate bool

	// Data represents the data that parsing from the config file. You can use it to access the
	// values in the config file.
	//
	// e.g. Data["foobar"] will accesses the value in the config file called "foobar".
	Data JSONMap
}

// defaultConfig is the default instance of the `Config`.
var defaultConfig = Config{
	AppName: "air",
	LogFormat: `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}","level":"{{.level}}",` +
		`"file":"{{.short_file}}","line":"{{.line}}"}`,
	Address:            "localhost:2333",
	TemplateRoot:       "templates",
	TemplateExt:        ".html",
	TemplateLeftDelim:  "{{",
	TemplateRightDelim: "}}",
}

// newConfig returns a pointer of a new instance of the `Config` by parsing the config file that in
// the rumtime directory named "config.yml" or "config.json". It returns the defaultConfig if the
// config file does not exist.
func newConfig() *Config {
	c := defaultConfig
	cfn := "config.yml"
	cfnJSON := "config.json"
	if _, err := os.Stat(cfn); err == nil || os.IsExist(err) {
		c.ParseFile(cfn)
	} else if _, err := os.Stat(cfnJSON); err == nil || os.IsExist(err) {
		c.ParseFile(cfnJSON)
	}
	return &c
}

// Parse parses the src into the c.
func (c *Config) Parse(src string) {
	if err := yaml.Unmarshal([]byte(src), &c.Data); err != nil {
		panic(err)
	}
	c.fillData()
}

// ParseFile parses the config file found in the filename path into the c.
func (c *Config) ParseFile(filename string) {
	if _, err := os.Stat(filename); err != nil && !os.IsExist(err) {
		panic(fmt.Sprintf("the config file %s does not exist", filename))
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	c.Parse(string(b))
}

// fillData fills the values of the fields from the field `Data` of the c.
func (c *Config) fillData() {
	if an, ok := c.Data["app_name"]; ok {
		c.AppName = an.(string)
	}
	if dm, ok := c.Data["debug_mode"]; ok {
		c.DebugMode = dm.(bool)
	}
	if le, ok := c.Data["log_enabled"]; ok {
		c.LogEnabled = le.(bool)
	}
	if lf, ok := c.Data["log_format"]; ok {
		c.LogFormat = lf.(string)
	}
	if addr, ok := c.Data["address"]; ok {
		c.Address = addr.(string)
	}
	if dh, ok := c.Data["disable_http2"]; ok {
		c.DisableHTTP2 = dh.(bool)
	}
	if tlscf, ok := c.Data["tls_cert_file"]; ok {
		c.TLSCertFile = tlscf.(string)
	}
	if tlskf, ok := c.Data["tls_key_file"]; ok {
		c.TLSKeyFile = tlskf.(string)
	}
	if rt, ok := c.Data["read_timeout"]; ok {
		c.ReadTimeout = time.Duration(rt.(int)) * time.Second
	}
	if wt, ok := c.Data["write_timeout"]; ok {
		c.WriteTimeout = time.Duration(wt.(int)) * time.Second
	}
	if tr, ok := c.Data["template_root"]; ok {
		c.TemplateRoot = tr.(string)
	}
	if te, ok := c.Data["template_ext"]; ok {
		c.TemplateExt = te.(string)
	}
	if tld, ok := c.Data["template_left_delim"]; ok {
		c.TemplateLeftDelim = tld.(string)
	}
	if trd, ok := c.Data["template_right_delim"]; ok {
		c.TemplateRightDelim = trd.(string)
	}
	if mt, ok := c.Data["minify_template"]; ok {
		c.MinifyTemplate = mt.(bool)
	}
}

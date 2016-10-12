package air

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

// Config is a global set of configurations that for an instance of `Air`
// for customization.
type Config struct {
	// AppName represens the name of current `Air` instance.
	//
	// Default Value is "air".
	//
	// It's called "app_name" in the config file.
	AppName string

	// DebugMode represents the state of `Air`'s debug mode.
	//
	// Default value is false.
	//
	// It's called "debug_mode" in the config file.
	DebugMode bool

	// LogFormat represents the format of `Logger`'s output content.
	//
	// Default value is:
	// `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}",` +
	// `"level":"{{.level}}","file":"{{.short_file}}","line":"{{.line}}"}`
	//
	// It's called "log_format" in the config file.
	LogFormat string

	// Address represents the TCP address that `Server` to listen
	// on.
	//
	// Default value is "localhost:8080".
	//
	// It's called "address" in the config file.
	Address string

	// Listener represens the custom `net.Listener`. If set, `Server`
	// accepts connections on it.
	//
	// Default value is nil.
	Listener net.Listener

	// TLSCertFile represents the TLS certificate file path.
	//
	// Default value is "".
	//
	// It's called "tls_cert_file" in the config file.
	TLSCertFile string

	// TLSKeyFile represents the TLS key file path.
	//
	// Default value is "".
	//
	// It's called "tls_key_file" in the config file.
	TLSKeyFile string

	// ReadTimeout represents the maximum duration before timing out
	// read of the request.
	//
	// Default value is 0.
	//
	// It's called "read_timeout" in the config file.
	//
	// *It's unit in the config file is SECONDS.*
	ReadTimeout time.Duration

	// WriteTimeout represents the maximum duration before timing
	// out write of the response.
	//
	// Default value is 0.
	//
	// It's called "write_timeout" in the config file.
	//
	// *It's unit in the config file is SECONDS.*
	WriteTimeout time.Duration

	// TemplatesRoot represents the root directory of the html templates.
	// It will be parsed into `Renderer`.
	//
	// Default value is "templates" that means a subdirectory of the
	// runtime directory.
	//
	// It's called "templates_root" in the config file.
	TemplatesRoot string

	// Data represents the data that parsing from config file. You can
	// use it to access the values in the config file.
	//
	// e.g. c.Data["mysql_user_name"] will accesses the value in config
	// file called "mysql_user_name".
	Data JSONMap
}

// defaultConfig is the default instance of `Config`.
var defaultConfig = Config{
	AppName: "air",
	LogFormat: `{"app_name":"{{.app_name}}","time":"{{.time_rfc3339}}",` +
		`"level":"{{.level}}","file":"{{.short_file}}","line":"{{.line}}"}`,
	Address:       "localhost:8080",
	TemplatesRoot: "templates",
}

// newConfig returns a new instance of `Config` by parsing the config file that
// in the rumtime directory named "config.json". It returns the defaultConfig
// if the config file doesn't exist.
func newConfig() *Config {
	c := defaultConfig
	cfn := "config.json"
	if _, err := os.Stat(cfn); err == nil || os.IsExist(err) {
		c.Parse(cfn)
	}
	return &c
}

// Parse parses the config file found in the filename path.
func (c *Config) Parse(filename string) {
	if _, err := os.Stat(filename); err != nil && !os.IsExist(err) {
		panic(fmt.Sprintf("config file %s does not exist", filename))
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &c.Data)
	if err != nil {
		panic(err)
	}

	c.fillData()
}

// fillData fills field's value from field `Data` of c.
func (c *Config) fillData() {
	if an, ok := c.Data["app_name"]; ok {
		c.AppName = an.(string)
	}
	if dm, ok := c.Data["debug_mode"]; ok {
		c.DebugMode = dm.(bool)
	}
	if lf, ok := c.Data["log_format"]; ok {
		c.LogFormat = lf.(string)
	}
	if addr, ok := c.Data["address"]; ok {
		c.Address = addr.(string)
	}
	if tlscf, ok := c.Data["tls_cert_file"]; ok {
		c.TLSCertFile = tlscf.(string)
	}
	if tlskf, ok := c.Data["tls_key_file"]; ok {
		c.TLSKeyFile = tlskf.(string)
	}
	if rt, ok := c.Data["read_timeout"]; ok {
		c.ReadTimeout = time.Duration(rt.(float64)) * time.Second
	}
	if wt, ok := c.Data["write_timeout"]; ok {
		c.WriteTimeout = time.Duration(wt.(float64)) * time.Second
	}
	if tr, ok := c.Data["templates_root"]; ok {
		c.TemplatesRoot = tr.(string)
	}
}

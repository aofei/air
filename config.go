package air

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"time"
)

// Config is a global set of configurations that for an instance of `Air`
// for customization.
type Config struct {
	// AppName represens the name of current `Air` instance.
	AppName string

	// JSON represents the JSON that parsing from config file.
	JSON map[string]interface{}

	// DebugMode represents the state of `Air`'s debug mode. Default
	// value is "false".
	DebugMode bool

	// LogFormat represents the format of `Logger`'s output content.
	// Default value is:
	// `{"app_name":"${app_name}","time":"${time_rfc3339}",` +
	// `"level":"${level}","file":"${short_file}","line":"${line}"}`
	LogFormat string

	// Address represents the TCP address that `Server` to listen
	// on. Default value is "localhost:8080".
	Address string

	// Listener represens the custom `net.Listener`. If set, `Server`
	// accepts connections on it. Default value is "nil".
	Listener net.Listener

	// TLSCertFile represents the TLS certificate file path. Default
	// value is "".
	TLSCertFile string

	// TLSKeyFile represents the TLS key file path. Default value
	// is "".
	TLSKeyFile string

	// ReadTimeout represents the maximum duration before timing out
	// read of the request. Default value is "0".
	ReadTimeout time.Duration

	// WriteTimeout represents the maximum duration before timing
	// out write of the response. Default value is "0".
	WriteTimeout time.Duration

	// TemplatesPath represents the path of `Renderer`'s templates.
	// Default value is "templates" that means a subdirectory of the
	// runtime directory.
	TemplatesPath string
}

// defaultConfig is the default instance of `Config`
var defaultConfig Config

// configJSON is the JSON map that parsing from config file
var configJSON map[string]interface{}

func init() {
	defaultConfig = Config{
		LogFormat: `{"app_name":"${app_name}","time":"${time_rfc3339}",` +
			`"level":"${level}","file":"${short_file}","line":"${line}"}`,
		Address:       "localhost:8080",
		TemplatesPath: "templates",
	}

	var cfn = "config.json"
	_, err := os.Stat(cfn)
	if err == nil || os.IsExist(err) {
		bytes, err := ioutil.ReadFile(cfn)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(bytes, &configJSON)
		if err != nil {
			panic(err)
		}
		if len(configJSON) == 0 {
			panic(errors.New("Need At Least One App In The Config File " +
				"Or Remove The Config File"))
		}
	}
}

// NewConfig returns a new instance of `Config` with a appName by parsing
// the config file that in the rumtime directory named "config.json".
// NewConfig returns the defaultConfig(field "AppName" be setted to provided
// appName) if the config file or appName doesn't exist.
func NewConfig(appName string) *Config {
	c := defaultConfig
	if configJSON == nil {
		c.AppName = appName
		return &c
	}

	if len(configJSON) == 1 {
		for k, v := range configJSON {
			c.AppName = k
			c.JSON = v.(map[string]interface{})
		}
	} else if configJSON[appName] == nil {
		panic(errors.New("App \"" + appName + "\" Not Exist"))
	} else {
		c.AppName = appName
		c.JSON = configJSON[appName].(map[string]interface{})
	}

	dm := c.JSON["debug_mode"]
	lf := c.JSON["log_format"]
	addr := c.JSON["address"]
	tlscf := c.JSON["tls_cert_file"]
	tlskf := c.JSON["tls_key_file"]
	rt := c.JSON["read_timeout"]
	wt := c.JSON["write_timeout"]
	tp := c.JSON["templates_path"]

	if dm != nil {
		c.DebugMode = dm.(bool)
	}
	if lf != nil {
		c.LogFormat = lf.(string)
	}
	if addr != nil {
		c.Address = addr.(string)
	}
	if tlscf != nil {
		c.TLSCertFile = tlscf.(string)
	}
	if tlskf != nil {
		c.TLSKeyFile = tlskf.(string)
	}
	if rt != nil {
		c.ReadTimeout = time.Duration(rt.(float64)) * time.Second
	}
	if wt != nil {
		c.WriteTimeout = time.Duration(wt.(float64)) * time.Second
	}
	if tp != nil {
		c.TemplatesPath = tp.(string)
	}
	return &c
}

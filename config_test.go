package air

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigParseYAML(t *testing.T) {
	c := newConfig()

	yaml := `app_name: "air"` + "\n" +
		`debug_mode: true` + "\n" +
		`log_enabled: true` + "\n" +
		`log_format: "air_log"` + "\n" +
		`address: "127.0.0.1:2333"` + "\n" +
		`disable_http2: true` + "\n" +
		`tls_cert_file: "path_to_tls_cert_file"` + "\n" +
		`tls_key_file: "path_to_tls_key_file"` + "\n" +
		`read_timeout: 60` + "\n" +
		`write_timeout: 60` + "\n" +
		`template_root: "ts"` + "\n" +
		`template_ext: ".tmpl"` + "\n" +
		`template_left_delim: "<<"` + "\n" +
		`template_right_delim: ">>"` + "\n" +
		`minify_template: true`

	c.Parse(yaml)
	assert.Equal(t, "air", c.AppName)
	assert.Equal(t, true, c.DebugMode)
	assert.Equal(t, true, c.LogEnabled)
	assert.Equal(t, "air_log", c.LogFormat)
	assert.Equal(t, "127.0.0.1:2333", c.Address)
	assert.Equal(t, true, c.DisableHTTP2)
	assert.Equal(t, "path_to_tls_cert_file", c.TLSCertFile)
	assert.Equal(t, "path_to_tls_key_file", c.TLSKeyFile)
	assert.Equal(t, 60*time.Second, c.ReadTimeout)
	assert.Equal(t, 60*time.Second, c.WriteTimeout)
	assert.Equal(t, "ts", c.TemplateRoot)
	assert.Equal(t, ".tmpl", c.TemplateExt)
	assert.Equal(t, "<<", c.TemplateLeftDelim)
	assert.Equal(t, ">>", c.TemplateRightDelim)
	assert.Equal(t, true, c.MinifyTemplate)
	assert.NotNil(t, c.Data)
}

func TestConfigParseJSON(t *testing.T) {
	c := newConfig()

	json := `{
			"app_name": "air",
			"debug_mode": true,
			"log_enabled": true,
			"log_format": "air_log",
			"address": "127.0.0.1:2333",
			"disable_http2": true,
			"tls_cert_file": "path_to_tls_cert_file",
			"tls_key_file": "path_to_tls_key_file",
			"read_timeout": 60,
			"write_timeout": 60,
			"template_root": "ts",
			"template_ext": ".tmpl",
			"template_left_delim": "<<",
			"template_right_delim": ">>",
			"minify_template": true
		 }`

	c.Parse(json)
	assert.Equal(t, "air", c.AppName)
	assert.Equal(t, true, c.DebugMode)
	assert.Equal(t, true, c.LogEnabled)
	assert.Equal(t, "air_log", c.LogFormat)
	assert.Equal(t, "127.0.0.1:2333", c.Address)
	assert.Equal(t, true, c.DisableHTTP2)
	assert.Equal(t, "path_to_tls_cert_file", c.TLSCertFile)
	assert.Equal(t, "path_to_tls_key_file", c.TLSKeyFile)
	assert.Equal(t, 60*time.Second, c.ReadTimeout)
	assert.Equal(t, 60*time.Second, c.WriteTimeout)
	assert.Equal(t, "ts", c.TemplateRoot)
	assert.Equal(t, ".tmpl", c.TemplateExt)
	assert.Equal(t, "<<", c.TemplateLeftDelim)
	assert.Equal(t, ">>", c.TemplateRightDelim)
	assert.Equal(t, true, c.MinifyTemplate)
	assert.NotNil(t, c.Data)
}

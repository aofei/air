package air

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigNewConfig(t *testing.T) {
	yaml := `app_name: "air"` + "\n" +
		`debug_mode: true` + "\n" +
		`log_enabled: true` + "\n" +
		`log_format: "air_log"` + "\n" +
		`address: "127.0.0.1:2333"` + "\n" +
		`tls_cert_file: "path_to_tls_cert_file"` + "\n" +
		`tls_key_file: "path_to_tls_key_file"` + "\n" +
		`read_timeout: 60` + "\n" +
		`write_timeout: 60` + "\n" +
		`template_root: "ts"` + "\n" +
		`template_ext: ".tmpl"` + "\n" +
		`template_left_delim: "<<"` + "\n" +
		`template_right_delim: ">>"` + "\n" +
		`minify_template: true`

	f, err := os.Create("config.yml")
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	f.WriteString(yaml)

	c := NewConfig("config.yml")

	assert.Equal(t, "air", c.AppName)
	assert.Equal(t, true, c.DebugMode)
	assert.Equal(t, true, c.LogEnabled)
	assert.Equal(t, "air_log", c.LogFormat)
	assert.Equal(t, "127.0.0.1:2333", c.Address)
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

func TestConfigParseError(t *testing.T) {
	c := &Config{}
	assert.Panics(t, func() {
		c.Parse("\t")
	})
}

func TestConfigParseFileError(t *testing.T) {
	c := &Config{}

	assert.Panics(t, func() {
		c.ParseFile("config_not_exist.yml")
	})

	dn := "config_is_a_directory.yml"
	err := os.Mkdir(dn, os.ModeDir)
	if err != nil {
		panic(err)
	}
	defer func() {
		os.Remove(dn)
	}()

	assert.Panics(t, func() {
		c.ParseFile(dn)
	})
}

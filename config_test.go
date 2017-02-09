package air

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigNewConfig(t *testing.T) {
	yaml := `
app_name: "air"
debug_mode: true
log_enabled: true
log_format: "air_log"
address: "127.0.0.1:2333"
tls_cert_file: "path_to_tls_cert_file"
tls_key_file: "path_to_tls_key_file"
read_timeout: 60
write_timeout: 60
template_root: "ts"
template_ext: ".tmpl"
template_left_delim: "<<"
template_right_delim: ">>"
template_minifed: true
`

	f, _ := os.Create("config.yml")
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	f.WriteString(yaml)

	c := NewConfig(f.Name())
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
	assert.Equal(t, true, c.TemplateMinified)
	assert.NotNil(t, c.Data)
}

func TestConfigParseError(t *testing.T) {
	c := &Config{}
	assert.Error(t, c.Parse("\t"))
}

func TestConfigParseFileError(t *testing.T) {
	c := &Config{}
	assert.Error(t, c.ParseFile("config_not_exist.yml"))
}

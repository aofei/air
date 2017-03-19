package air

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigNewConfig(t *testing.T) {
	toml := `
app_name = "air"
debug_mode = true
logger_enabled = true
log_format = "air_log"
address = "127.0.0.1:2333"
read_timeout = 200
write_timeout = 200
max_header_bytes = 65536
tls_cert_file = "path_to_tls_cert_file"
tls_key_file = "path_to_tls_key_file"
template_root = "ts"
template_exts = [".tmpl"]
template_left_delim = "<<"
template_right_delim = ">>"
template_minified = true
coffer_enabled = true
asset_root = "as"
asset_exts = [".jpg"]
asset_minified = true
`

	f, _ := os.Create("config.toml")
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	f.WriteString(toml)

	c := NewConfig(f.Name())
	assert.Equal(t, "air", c.AppName)
	assert.Equal(t, true, c.DebugMode)
	assert.Equal(t, true, c.LoggerEnabled)
	assert.Equal(t, "air_log", c.LogFormat)
	assert.Equal(t, "127.0.0.1:2333", c.Address)
	assert.Equal(t, 200*time.Millisecond, c.ReadTimeout)
	assert.Equal(t, 200*time.Millisecond, c.WriteTimeout)
	assert.Equal(t, 65536, c.MaxHeaderBytes)
	assert.Equal(t, "path_to_tls_cert_file", c.TLSCertFile)
	assert.Equal(t, "path_to_tls_key_file", c.TLSKeyFile)
	assert.Equal(t, "ts", c.TemplateRoot)
	assert.Equal(t, []string{".tmpl"}, c.TemplateExts)
	assert.Equal(t, "<<", c.TemplateLeftDelim)
	assert.Equal(t, ">>", c.TemplateRightDelim)
	assert.Equal(t, true, c.TemplateMinified)
	assert.Equal(t, true, c.CofferEnabled)
	assert.Equal(t, "as", c.AssetRoot)
	assert.Equal(t, []string{".jpg"}, c.AssetExts)
	assert.Equal(t, true, c.AssetMinified)
	assert.NotNil(t, c.Data)
}

func TestConfigParseError(t *testing.T) {
	c := &Config{}
	assert.Error(t, c.Parse("[air"))
}

func TestConfigParseFileError(t *testing.T) {
	c := &Config{}
	assert.Error(t, c.ParseFile("config_not_exist.toml"))
}

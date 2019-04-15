package air

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewI18n(t *testing.T) {
	a := New()
	i := a.i18n

	assert.NotNil(t, i)
	assert.NotNil(t, i.a)
	assert.NotNil(t, i.loadOnce)
	assert.Nil(t, i.watcher)
	assert.Nil(t, i.matcher)
	assert.Nil(t, i.locales)
}

func TestI18nLoad(t *testing.T) {
	a := New()
	a.I18nEnabled = true

	dir, err := ioutil.TempDir("", "air.TestI18nLoad")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.I18nLocaleRoot = dir

	i := a.i18n

	i.load()
	assert.Nil(t, i.loadError)
	assert.NotNil(t, i.watcher)
	assert.NotNil(t, i.matcher)
	assert.NotNil(t, i.locales)
}

func TestI18nLocalize(t *testing.T) {
	a := New()
	a.I18nEnabled = true

	dir, err := ioutil.TempDir("", "air.TestI18nLocalize")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.I18nLocaleRoot = dir

	i := a.i18n

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.I18nLocaleRoot, "en-US.toml"),
		[]byte(`"Foobar" = "Foobar"`),
		os.ModePerm,
	))

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.I18nLocaleRoot, "en-GB.toml"),
		nil,
		os.ModePerm,
	))

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.I18nLocaleRoot, "de-DE.ext"),
		[]byte(`"Foobar" = "Fubar"`),
		os.ModePerm,
	))

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.I18nLocaleRoot, "zh-CN.toml"),
		[]byte(`"Foobar" = "测试"`),
		os.ModePerm,
	))

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	i.localize(req)
	assert.NotNil(t, req.localizedString)
	assert.Equal(t, "Foobar", req.LocalizedString("Foobar"))
	assert.Equal(t, "Barfoo", req.LocalizedString("Barfoo"))

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-GB")

	i.localize(req)
	assert.NotNil(t, req.localizedString)
	assert.Equal(t, "Foobar", req.LocalizedString("Foobar"))

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "de-DE")

	i.localize(req)
	assert.NotNil(t, req.localizedString)
	assert.Equal(t, "Foobar", req.LocalizedString("Foobar"))

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "zh-CN")

	i.localize(req)
	assert.NotNil(t, req.localizedString)
	assert.Equal(t, "测试", req.LocalizedString("Foobar"))

	a = New()
	i = a.i18n

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/", nil)

	log.SetOutput(ioutil.Discard)
	i.localize(req)
	log.SetOutput(os.Stderr)

	assert.Error(t, i.loadError)
}

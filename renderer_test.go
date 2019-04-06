package air

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRenderer(t *testing.T) {
	a := New()
	r := a.renderer

	assert.NotNil(t, r)
	assert.NotNil(t, r.a)
	assert.NotNil(t, r.loadOnce)
	assert.Nil(t, r.watcher)
	assert.Nil(t, r.template)
}

func TestRendererLoad(t *testing.T) {
	a := New()

	dir, err := ioutil.TempDir("", "air.TestRendererLoad")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.RendererTemplateRoot = dir

	r := a.renderer

	r.load()
	assert.Nil(t, r.loadError)
	assert.NotNil(t, r.watcher)
	assert.NotNil(t, r.template)
}

func TestRendererRender(t *testing.T) {
	a := New()

	dir, err := ioutil.TempDir("", "air.TestRendererRender")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.RendererTemplateRoot = dir

	r := a.renderer

	assert.NoError(t, ioutil.WriteFile(
		path.Join(a.RendererTemplateRoot, "test.html"),
		[]byte(`<a href="/">Go Home</a>`),
		os.ModePerm,
	))

	assert.NoError(t, ioutil.WriteFile(
		path.Join(a.RendererTemplateRoot, "test.ext"),
		[]byte(`<a href="/">Go Home Again</a>`),
		os.ModePerm,
	))

	assert.NoError(t, r.render(ioutil.Discard, "test.html", nil, locstr))
	assert.Error(t, r.render(ioutil.Discard, "test.ext", nil, locstr))

	a.I18nEnabled = true
	assert.Error(t, r.render(ioutil.Discard, "test.html", nil, locstr))

	a = New()
	a.I18nEnabled = true
	a.RendererTemplateRoot = dir

	r = a.renderer

	assert.NoError(t, r.render(ioutil.Discard, "test.html", nil, locstr))
}

func TestStrlen(t *testing.T) {
	assert.Equal(t, 6, strlen("Foobar"))
	assert.Equal(t, 2, strlen("测试"))
}

func TestSubstr(t *testing.T) {
	assert.Equal(t, "o", substr("Foobar", 1, 2))
	assert.Equal(t, "试", substr("测试", 1, 2))
}

func TestTimefmt(t *testing.T) {
	assert.Equal(
		t,
		"1970-01-01T00:00:00Z",
		timefmt(time.Unix(0, 0).UTC(), time.RFC3339),
	)
}

func TestLocstr(t *testing.T) {
	assert.Equal(t, "Foobar", locstr("Foobar"))
}

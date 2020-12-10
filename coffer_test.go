package air

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/stretchr/testify/assert"
)

func TestNewCoffer(t *testing.T) {
	a := New()
	c := a.coffer

	assert.NotNil(t, c)
	assert.NotNil(t, c.a)
	assert.NotNil(t, c.loadOnce)
	assert.Nil(t, c.watcher)
	assert.Nil(t, c.cache)
}

func TestCofferLoad(t *testing.T) {
	a := New()
	c := a.coffer

	c.load()
	assert.Nil(t, c.loadError)
	assert.NotNil(t, c.watcher)
	assert.NotNil(t, c.cache)
}

func TestCofferAsset(t *testing.T) {
	a := New()
	a.MinifierEnabled = true
	a.GzipEnabled = true
	a.GzipMinContentLength = 0

	dir, err := ioutil.TempDir("", "air.TestCofferAsset")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.CofferAssetRoot = dir

	c := a.coffer

	a1, err := c.asset(filepath.Join(a.CofferAssetRoot, "test.html"))
	assert.Error(t, err)
	assert.Nil(t, a1)

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.CofferAssetRoot, "test.html"),
		[]byte(`<a href="/">Go Home</a>`),
		os.ModePerm,
	))

	a2, err := c.asset(filepath.Join(a.CofferAssetRoot, "test.html"))
	assert.NoError(t, err)
	assert.NotNil(t, a2)

	a3, err := c.asset(filepath.Join(a.CofferAssetRoot, "test.html"))
	assert.NoError(t, err)
	assert.NotNil(t, a3)

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.CofferAssetRoot, "test.html"),
		[]byte(`<a href="/">Go Home Again</a>`),
		os.ModePerm,
	))

	a4, err := c.asset(filepath.Join(a.CofferAssetRoot, "test.html"))
	assert.NoError(t, err)
	assert.NotNil(t, a4)

	a5, err := c.asset("test.html")
	assert.NoError(t, err)
	assert.Nil(t, a5)

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.CofferAssetRoot, "test.ext"),
		[]byte(`<a href="/">Go Home</a>`),
		os.ModePerm,
	))

	a6, err := c.asset(filepath.Join(a.CofferAssetRoot, "test.ext"))
	assert.NoError(t, err)
	assert.Nil(t, a6)
}

func TestAssetContent(t *testing.T) {
	a := New()
	a.MinifierEnabled = true
	a.GzipEnabled = true
	a.GzipMinContentLength = 0

	dir, err := ioutil.TempDir("", "air.TestCofferAsset")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.CofferAssetRoot = dir

	c := a.coffer

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.CofferAssetRoot, "test.html"),
		[]byte(`<a href="/">Go Home</a>`),
		os.ModePerm,
	))

	a1, err := c.asset(filepath.Join(a.CofferAssetRoot, "test.html"))
	assert.NoError(t, err)
	assert.NotNil(t, a1)

	b := a1.content(false)
	assert.Equal(t, "<a href=/>Go Home</a>", string(b))

	b = a1.content(true)
	assert.NotNil(t, b)

	c.cache = fastcache.New(c.a.CofferMaxMemoryBytes)

	b = a1.content(false)
	assert.Nil(t, b)
}

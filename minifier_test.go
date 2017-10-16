package air

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinifierInit(t *testing.T) {
	a := New()
	a.Minifier.Init()

	b, err := a.Minifier.Minify("text/html", []byte("<air></air>"))
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("text/css", []byte(".air{}"))
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("text/javascript", []byte("alert('air')"))
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("application/json", []byte("{}"))
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("text/xml", []byte("<air></air>"))
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("image/svg+xml", []byte("<air></air>"))
	assert.NotNil(t, b)
	assert.NoError(t, err)
}

func TestMinifierMinify(t *testing.T) {
	a := New()
	a.Minifier.Init()

	m := image.NewRGBA(image.Rect(0, 0, 1, 1))
	c := color.RGBA{0, 0, 0, 0}
	draw.Draw(m, m.Bounds(), &image.Uniform{c}, image.ZP, draw.Src)

	j, _ := os.Create("air.jpg")
	defer func() {
		os.Remove(j.Name())
	}()
	jpeg.Encode(j, m, nil)
	j.Close()

	p, _ := os.Create("air.png")
	defer func() {
		os.Remove(p.Name())
	}()
	png.Encode(p, m)
	p.Close()

	b, err := a.Minifier.Minify("image/jpeg", []byte("encoding error"))
	assert.Nil(t, b)
	assert.Error(t, err)

	b, _ = ioutil.ReadFile(j.Name())
	b, err = a.Minifier.Minify("image/jpeg", b)
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("image/png", []byte("encoding error"))
	assert.Nil(t, b)
	assert.Error(t, err)

	b, _ = ioutil.ReadFile(p.Name())
	b, err = a.Minifier.Minify("image/png", b)
	assert.NotNil(t, b)
	assert.NoError(t, err)

	b, err = a.Minifier.Minify("text/css", []byte("error"))
	assert.Nil(t, b)
	assert.Error(t, err)

	b, err = a.Minifier.Minify("unsupported", []byte("unsupported"))
	assert.Nil(t, b)
	assert.Error(t, err)
}

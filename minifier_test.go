package air

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinifier(t *testing.T) {
	// Singleton

	assert.NotNil(t, minifierSingleton)
	assert.NotNil(t, minifierSingleton.minifier)

	// HTML

	b, err := minifierSingleton.minify(
		"text/html",
		[]byte("<!DOCTYPE html>"),
	)
	assert.Equal(t, "<!doctype html>", string(b))
	assert.NoError(t, err)

	// HTML with charset

	b, err = minifierSingleton.minify(
		"text/html; charset=utf-8",
		[]byte("<!DOCTYPE html>"),
	)
	assert.Equal(t, "<!doctype html>", string(b))
	assert.NoError(t, err)

	// CSS

	b, err = minifierSingleton.minify(
		"text/css",
		[]byte("body { font-size: 16px; }"),
	)
	assert.Equal(t, "body{font-size:16px}", string(b))
	assert.NoError(t, err)

	// JavaScript

	b, err = minifierSingleton.minify(
		"text/javascript",
		[]byte("var foo = \"bar\";"),
	)
	assert.Equal(t, "var foo=\"bar\";", string(b))
	assert.NoError(t, err)

	// JSON

	b, err = minifierSingleton.minify(
		"application/json",
		[]byte("{ \"foo\": \"bar\" }"),
	)
	assert.Equal(t, "{\"foo\":\"bar\"}", string(b))
	assert.NoError(t, err)

	// XML

	b, err = minifierSingleton.minify(
		"text/xml",
		[]byte("<Foobar></Foobar>"),
	)
	assert.Equal(t, "<Foobar/>", string(b))
	assert.NoError(t, err)

	// SVG

	b, err = minifierSingleton.minify(
		"image/svg+xml",
		[]byte("<Foobar></Foobar>"),
	)
	assert.Equal(t, "<Foobar/>", string(b))
	assert.NoError(t, err)

	// JPEG

	buf := &bytes.Buffer{}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	draw.Draw(
		img,
		img.Bounds(),
		image.NewUniform(color.RGBA{0, 0, 0, 0}),
		image.ZP,
		draw.Src,
	)

	jpeg.Encode(buf, img, nil)

	b, err = minifierSingleton.minify("image/jpeg", buf.Bytes())
	assert.NotEmpty(t, b)
	assert.NoError(t, err)

	// PNG

	buf.Reset()
	png.Encode(buf, img)

	b, err = minifierSingleton.minify("image/png", buf.Bytes())
	assert.NotEmpty(t, b)
	assert.NoError(t, err)

	// Errors

	b, err = minifierSingleton.minify("application/json", []byte("{:}"))
	assert.Nil(t, b)
	assert.Error(t, err)

	b, err = minifierSingleton.minify("image/jpeg", nil)
	assert.Nil(t, b)
	assert.Error(t, err)

	b, err = minifierSingleton.minify("image/png", nil)
	assert.Nil(t, b)
	assert.Error(t, err)

	b, err = minifierSingleton.minify("unsupported", nil)
	assert.Nil(t, b)
	assert.Error(t, err)
}

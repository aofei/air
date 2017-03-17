package air

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinifierInit(t *testing.T) {
	a := New()
	a.Minifier.Init()

	_, err := a.Minifier.Minify(MIMETextHTML, []byte{})
	assert.NoError(t, err)

	_, err = a.Minifier.Minify(MIMETextCSS, []byte{})
	assert.NoError(t, err)

	_, err = a.Minifier.Minify(MIMETextJavaScript, []byte{})
	assert.NoError(t, err)

	_, err = a.Minifier.Minify(MIMEApplicationJSON, []byte{})
	assert.NoError(t, err)

	_, err = a.Minifier.Minify(MIMETextXML, []byte{})
	assert.NoError(t, err)

	_, err = a.Minifier.Minify(MIMEImageSVGXML, []byte{})
	assert.NoError(t, err)
}

func TestMinifierMinifyError(t *testing.T) {
	a := New()
	a.Minifier.Init()

	b, err := a.Minifier.Minify("error", []byte{})

	assert.Nil(t, b)
	assert.Error(t, err)
}

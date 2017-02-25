package air

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinifierInit(t *testing.T) {
	a := New()
	a.Minifier.Init()

	w := &bytes.Buffer{}
	r := &bytes.Reader{}

	assert.NoError(t, a.Minifier.Minify(MIMETextHTML, w, r))
	assert.NoError(t, a.Minifier.Minify(MIMETextCSS, w, r))
	assert.NoError(t, a.Minifier.Minify(MIMETextJavaScript, w, r))
	assert.NoError(t, a.Minifier.Minify(MIMEApplicationJSON, w, r))
	assert.NoError(t, a.Minifier.Minify(MIMETextXML, w, r))
	assert.NoError(t, a.Minifier.Minify(MIMEImageSVG, w, r))
}

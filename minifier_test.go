package air

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMinifier(t *testing.T) {
	a := New()
	m := a.minifier

	assert.NotNil(t, m)
	assert.NotNil(t, m.a)
	assert.Nil(t, m.minifier)
}

func TestMinifierMinify(t *testing.T) {
	a := New()
	m := a.minifier

	b, err := m.minify("", nil)
	assert.NoError(t, err)
	assert.Empty(t, string(b))

	b, err = m.minify("text/html", []byte("<a href=\"/\">Go Home</a>"))
	assert.NoError(t, err)
	assert.Equal(t, "<a href=/>Go Home</a>", string(b))
}

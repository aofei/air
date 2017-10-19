package air

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURL(t *testing.T) {
	su, err := url.Parse("https://example.com/foo/bar?foo=bar#foobar")
	assert.NotNil(t, su)
	assert.Nil(t, err)

	u := newURL(su)
	assert.Equal(t, su.Scheme, u.Scheme)
	assert.Equal(t, su.Host, u.Host)
	assert.Equal(t, su.EscapedPath(), u.Path)
	assert.Equal(t, su.RawQuery, u.Query)
	assert.Equal(t, su.Fragment, u.Fragment)
	assert.Equal(t, su.String(), u.String())
}

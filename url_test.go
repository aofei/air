package air

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURL(t *testing.T) {
	su, _ := url.ParseRequestURI("https://example.com/foo/bar?foo=bar")

	u := &URL{
		Scheme: su.Scheme,
		Host:   su.Host,
		Path:   su.EscapedPath(),
		Query:  su.RawQuery,
	}

	assert.Equal(t, su.Scheme, u.Scheme)
	assert.Equal(t, su.Host, u.Host)
	assert.Equal(t, su.EscapedPath(), u.Path)
	assert.Equal(t, su.RawQuery, u.Query)
	assert.Equal(t, su.String(), u.String())

	u.Path = u.Path[1:]
	assert.Equal(t, su.String(), u.String())
}

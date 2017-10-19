package air

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	a := New()
	sr := httptest.NewRequest(
		"GET",
		"https://example.com/foo/bar?foo=bar#foobar",
		bytes.NewBufferString("foobar"),
	)

	r := newRequest(a, sr)
	assert.Equal(t, a, r.air)
	assert.Equal(t, sr, r.request)
	assert.Equal(t, sr.Method, r.Method)
	assert.Equal(t, sr.URL.String(), r.URL.String())
	assert.Equal(t, sr.Proto, r.Proto)
	assert.Equal(t, len(sr.Header), len(r.Headers))
	assert.Equal(t, sr.Body, r.Body)
	assert.Equal(t, len(sr.Cookies()), len(r.Cookies))
	assert.Zero(t, len(r.PathParams))
	assert.Equal(t, len(sr.URL.Query()), len(r.QueryParams))
	assert.Equal(t, len(sr.Form), len(r.FormParams))
	assert.Zero(t, len(r.FormFiles))
	assert.Zero(t, len(r.Values))
}

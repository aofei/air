package air

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	a := New()
	sr := httptest.NewRequest(
		"POST",
		"https://example.com/foo/bar?foo=bar#foobar",
		bytes.NewBufferString(`{"Foobar":"Foobar"}`),
	)
	sr.Header.Set("Content-Type", "application/json")
	sr.Header.Set("Set-Cookie", "foo=bar")
	sr.MultipartForm = &multipart.Form{
		File: map[string][]*multipart.FileHeader{
			"foobar": []*multipart.FileHeader{
				&multipart.FileHeader{},
			},
		},
	}

	r := newRequest(a, sr)
	assert.NotNil(t, r)
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
	assert.Equal(t, len(sr.MultipartForm.File), len(r.FormFiles))
	assert.Zero(t, len(r.Values))

	var s struct {
		Foobar string
	}

	assert.Nil(t, r.Bind(&s))
	assert.Equal(t, "Foobar", s.Foobar)
}

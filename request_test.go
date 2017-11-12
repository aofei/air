package air

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	sr := httptest.NewRequest(
		"POST",
		"https://example.com/foo/bar?foo=bar#foobar",
		bytes.NewBufferString(`{"Foobar":"Foobar"}`),
	)
	sr.Header.Set("Content-Type", "application/json")
	sr.Header.Set("Cookie", "foo=bar")

	fileHeader := &multipart.FileHeader{}
	rs := reflect.ValueOf(fileHeader).Elem()
	rf := rs.FieldByName("content")
	rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
	rf.Set(reflect.ValueOf([]byte{}))

	sr.MultipartForm = &multipart.Form{
		File: map[string][]*multipart.FileHeader{
			"foobar": {
				fileHeader,
			},
		},
	}

	r := newRequest(sr)
	assert.NotNil(t, r)
	assert.Equal(t, sr.Method, r.Method)
	assert.Equal(t, sr.URL.String(), r.URL.String())
	assert.Equal(t, sr.Proto, r.Proto)
	assert.NotNil(t, r.Headers)
	assert.Equal(t, len(sr.Header), len(r.Headers))
	assert.Equal(t, sr.Body, r.Body)
	assert.Equal(t, len(sr.Cookies()), len(r.Cookies))
	assert.NotNil(t, r.Params)
	assert.Equal(t, len(sr.Form), len(r.Params))
	assert.NotNil(t, r.Files)
	assert.Equal(t, len(sr.MultipartForm.File), len(r.Files))
	assert.Equal(t, sr.RemoteAddr, r.RemoteAddr)
	assert.NotNil(t, r.Values)
	assert.Zero(t, len(r.Values))
	assert.Equal(t, sr, r.request)

	var s struct {
		Foobar string
	}

	assert.NoError(t, r.Bind(&s))
	assert.Equal(t, "Foobar", s.Foobar)
}

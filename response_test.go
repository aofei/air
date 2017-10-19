package air

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com/foo/bar?foo=bar#foobar",
		bytes.NewBufferString("foobar"),
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.Equal(t, a, r.air)
	assert.Equal(t, req, r.request)
	assert.Equal(t, rec, r.writer)
	assert.Equal(t, rec, r.flusher)
	assert.Nil(t, r.hijacker)
	assert.Nil(t, r.closeNotifier)
	assert.Nil(t, r.pusher)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Zero(t, r.Size)
	assert.False(t, r.Written)
}

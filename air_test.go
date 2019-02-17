package air

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSliceContains(t *testing.T) {
	assert.True(t, stringSliceContains([]string{"foo"}, "foo"))
	assert.False(t, stringSliceContains([]string{"foo"}, "bar"))
}

func TestStringSliceContainsCIly(t *testing.T) {
	assert.True(t, stringSliceContainsCIly([]string{"foo"}, "FOO"))
	assert.False(t, stringSliceContainsCIly([]string{"foo"}, "BAR"))
}

func TestSplitPathQuery(t *testing.T) {
	p, q := splitPathQuery("/foobar")
	assert.Equal(t, "/foobar", p)
	assert.Empty(t, q)

	p, q = splitPathQuery("/foobar?")
	assert.Equal(t, "/foobar", p)
	assert.Empty(t, q)

	p, q = splitPathQuery("/foobar?foo=bar")
	assert.Equal(t, "/foobar", p)
	assert.Equal(t, "foo=bar", q)
}

func fakeRRCycle(
	a *Air,
	method string,
	target string,
	body io.Reader,
) (*Request, *Response, *httptest.ResponseRecorder) {
	req := &Request{
		Air: a,

		parseRouteParamsOnce: &sync.Once{},
		parseOtherParamsOnce: &sync.Once{},
	}
	req.SetHTTPRequest(httptest.NewRequest(method, target, body))

	rec := httptest.NewRecorder()
	res := &Response{
		Air:    a,
		Status: http.StatusOK,

		req:  req,
		ohrw: rec,
	}
	res.SetHTTPResponseWriter(&responseWriter{
		r: res,
		w: rec,
	})

	req.res = res

	return req, res, rec
}

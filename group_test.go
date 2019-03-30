package air

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	a := New()
	g := a.Group("/foo")

	assert.NotNil(t, g)
	assert.Equal(t, a, g.Air)
	assert.Equal(t, "/foo", g.Prefix)
	assert.Nil(t, g.Gases)

	g.GET("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [GET /foo/bar]")
	})

	g.HEAD("/bar", func(req *Request, res *Response) error {
		return res.Write(nil)
	})

	g.POST("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [POST /foo/bar]")
	})

	g.PUT("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [PUT /foo/bar]")
	})

	g.PATCH("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [PATCH /foo/bar]")
	})

	g.DELETE("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [DELETE /foo/bar]")
	})

	g.CONNECT("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [CONNECT /foo/bar]")
	})

	g.OPTIONS("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [OPTIONS /foo/bar]")
	})

	g.TRACE("/bar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [TRACE /foo/bar]")
	})

	g.BATCH(nil, "/bar2", func(req *Request, res *Response) error {
		return res.WriteString("Matched [* /foo/bar2]")
	})

	dir, err := ioutil.TempDir("", "air.TestGroup")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	f, err := ioutil.TempFile(dir, "")
	assert.NoError(t, err)
	assert.NotNil(t, f)

	_, err = f.Write([]byte("Foobar"))
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	f2, err := ioutil.TempFile(dir, "")
	assert.NoError(t, err)
	assert.NotNil(t, f2)

	_, err = f2.Write([]byte("Foobar2"))
	assert.NoError(t, err)
	assert.NoError(t, f2.Close())

	g.FILE("/bar3", f.Name())
	g.FILES("/bar4", dir)

	g2 := g.Group("/bar5")
	assert.NotNil(t, g2)
	assert.Equal(t, a, g2.Air)
	assert.Equal(t, "/foo/bar5", g2.Prefix)
	assert.Nil(t, g2.Gases)

	req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [GET /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodPost, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [POST /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPut, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [PUT /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPatch, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [PATCH /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodDelete, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [DELETE /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodConnect, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [CONNECT /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodOptions, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [OPTIONS /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodTrace, "/foo/bar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [TRACE /foo/bar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodPost, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPut, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPatch, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodDelete, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodConnect, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodOptions, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodTrace, "/foo/bar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foo/bar2]", rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/foo/bar4", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "Not Found", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foo/bar4", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/foo/bar5", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "Not Found", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foo/bar5", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/foo/bar3", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foo/bar3", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(
		http.MethodGet,
		path.Join("/foo/bar4", filepath.Base(f2.Name())),
		nil,
	)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar2", rec.Body.String())

	req = httptest.NewRequest(
		http.MethodHead,
		path.Join("/foo/bar4/", filepath.Base(f2.Name())),
		nil,
	)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

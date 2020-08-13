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

	hr := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	hrw := httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr := hrw.Result()
	hrwrb, _ := ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [GET /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodHead, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Len(t, hrwrb, 0)

	hr = httptest.NewRequest(http.MethodPost, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [POST /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodPut, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [PUT /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodPatch, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [PATCH /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodDelete, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [DELETE /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodConnect, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [CONNECT /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodOptions, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [OPTIONS /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodTrace, "/foo/bar", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [TRACE /foo/bar]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodGet, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodHead, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Len(t, hrwrb, 0)

	hr = httptest.NewRequest(http.MethodPost, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodPut, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodPatch, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodDelete, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodConnect, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodOptions, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodTrace, "/foo/bar2", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Matched [* /foo/bar2]", string(hrwrb))

	hr = httptest.NewRequest(http.MethodGet, "/foo/bar4", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusNotFound, hrwr.StatusCode)
	assert.Equal(t, "Not Found", string(hrwrb))

	hr = httptest.NewRequest(http.MethodHead, "/foo/bar4", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusNotFound, hrwr.StatusCode)
	assert.Len(t, hrwrb, 0)

	hr = httptest.NewRequest(http.MethodGet, "/foo/bar5", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusNotFound, hrwr.StatusCode)
	assert.Equal(t, "Not Found", string(hrwrb))

	hr = httptest.NewRequest(http.MethodHead, "/foo/bar5", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusNotFound, hrwr.StatusCode)
	assert.Len(t, hrwrb, 0)

	hr = httptest.NewRequest(http.MethodGet, "/foo/bar3", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Foobar", string(hrwrb))

	hr = httptest.NewRequest(http.MethodHead, "/foo/bar3", nil)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Len(t, hrwrb, 0)

	hr = httptest.NewRequest(
		http.MethodGet,
		path.Join("/foo/bar4", filepath.Base(f2.Name())),
		nil,
	)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Equal(t, "Foobar2", string(hrwrb))

	hr = httptest.NewRequest(
		http.MethodHead,
		path.Join("/foo/bar4/", filepath.Base(f2.Name())),
		nil,
	)
	hrw = httptest.NewRecorder()

	a.ServeHTTP(hrw, hr)

	hrwr = hrw.Result()
	hrwrb, _ = ioutil.ReadAll(hrwr.Body)

	assert.Equal(t, http.StatusOK, hrwr.StatusCode)
	assert.Len(t, hrwrb, 0)
}

package air

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupContain(t *testing.T) {
	a := New()
	a.server = newServer(a)
	g := NewGroup(a, "/group")

	g.Contain(WrapGas(func(c *Context) error {
		return c.String("group gas")
	}))

	g.GET("/", func(*Context) error { return nil })

	req, _ := http.NewRequest(GET, "/group", nil)
	rec := httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, "group gas", rec.Body.String())
}

func TestGroupRESTfulMethods(t *testing.T) {
	a := New()
	g := NewGroup(a, "/group")
	h := func(*Context) error { return nil }

	g.GET("/", h)
	g.POST("/", h)
	g.PUT("/", h)
	g.DELETE("/", h)
}

func TestGroupStatic(t *testing.T) {
	a := New()
	a.server = newServer(a)

	prefix := "/group"
	secondPrefix := "/air"
	fn := "air.go"

	g := NewGroup(a, prefix)
	g.Static(secondPrefix, "./")

	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest(GET, prefix+secondPrefix+"/"+fn, nil)
	rec := httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())

	fn = "air_test.go"

	b, _ = ioutil.ReadFile(fn)
	req, _ = http.NewRequest(GET, prefix+secondPrefix+"/"+fn, nil)
	rec = httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestGroupFile(t *testing.T) {
	a := New()
	a.server = newServer(a)

	prefix := "/group"
	secondPrefix := "/group2"
	path := "/air"
	fn := "air.go"

	g := NewGroup(a, prefix)
	sg := g.NewSubGroup(secondPrefix)
	sg.File(path, fn)

	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest(GET, prefix+secondPrefix+path, nil)
	rec := httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

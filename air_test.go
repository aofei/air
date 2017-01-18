package air

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAirNew(t *testing.T) {
	a := New()

	assert.Equal(t, 0, len(a.pregases))
	assert.Equal(t, 0, len(a.gases))
	assert.Nil(t, a.server)
	assert.NotNil(t, a.router)

	assert.NotNil(t, a.Config)
	assert.NotNil(t, a.Logger)
	assert.NotNil(t, a.Binder)
	assert.NotNil(t, a.Renderer)
	assert.NotNil(t, a.HTTPErrorHandler)

	assert.NotNil(t, contextPool)
}

func TestAirPrecontain(t *testing.T) {
	a := New()
	a.server = newServer(a)
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()

	pregas := WrapGas(func(c *Context) error { return c.String("pregas") })

	a.Precontain(pregas)
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, "pregas", rec.Body.String())
}

func TestAirContain(t *testing.T) {
	a := New()
	a.server = newServer(a)
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()

	gas := WrapGas(func(c *Context) error { return c.String("gas") })

	a.Contain(gas)
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, "gas", rec.Body.String())
}

func TestAirMethods(t *testing.T) {
	a := New()
	a.server = newServer(a)
	path := "/methods"
	req, _ := http.NewRequest(GET, path, nil)
	rec := httptest.NewRecorder()

	a.GET(path, func(c *Context) error { return c.String(GET) })
	a.POST(path, func(c *Context) error { return c.String(POST) })
	a.PUT(path, func(c *Context) error { return c.String(PUT) })
	a.DELETE(path, func(c *Context) error { return c.String(DELETE) })

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, GET, rec.Body.String())

	req.Method = POST
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, POST, rec.Body.String())

	req.Method = PUT
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, PUT, rec.Body.String())

	req.Method = DELETE
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, DELETE, rec.Body.String())
}

func TestAirStatic(t *testing.T) {
	a := New()
	a.server = newServer(a)
	prefix := "/air"
	fn := "air.go"
	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest(GET, prefix+"/"+fn, nil)
	rec := httptest.NewRecorder()

	a.Static(prefix, "./")

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())

	fn = "air_test.go"
	b, _ = ioutil.ReadFile(fn)
	req, _ = http.NewRequest(GET, prefix+"/"+fn, nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestAirFile(t *testing.T) {
	a := New()
	a.server = newServer(a)
	path := "/air"
	fn := "air.go"
	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest(GET, path, nil)
	rec := httptest.NewRecorder()

	a.File(path, fn)

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestAirURL(t *testing.T) {
	a := New()
	h := func(c *Context) error { return c.NoContent() }
	a.GET("/:first/:second", h)
	assert.Equal(t, "/foo/bar", a.URL(h, "foo", "bar"))
}

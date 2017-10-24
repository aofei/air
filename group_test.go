package air

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupRESTfulMethods(t *testing.T) {
	a := New()
	g := &Group{
		Air:    a,
		Prefix: "/group",
	}

	g.GET("/", nil)
	g.HEAD("/", nil)
	g.POST("/", nil)
	g.PUT("/", nil)
	g.PATCH("/", nil)
	g.DELETE("/", nil)
	g.CONNECT("/", nil)
	g.OPTIONS("/", nil)
	g.TRACE("/", nil)
}

func TestGroupSTATIC(t *testing.T) {
	a := New()
	a.server = newServer(a)

	prefix := "/group"
	secondPrefix := "/air"
	fn := "air.go"

	g := &Group{
		Air:    a,
		Prefix: prefix,
	}
	g.STATIC(secondPrefix, ".")

	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest("GET", prefix+secondPrefix+"/"+fn, nil)
	rec := httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())

	fn = "air_test.go"

	b, _ = ioutil.ReadFile(fn)
	req, _ = http.NewRequest("GET", prefix+secondPrefix+"/"+fn, nil)
	rec = httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestGroupFILE(t *testing.T) {
	a := New()
	a.server = newServer(a)

	prefix := "/group"
	path := "/air"
	fn := "air.go"

	g := &Group{
		Air:    a,
		Prefix: prefix,
	}
	g.FILE(path, fn)

	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest("GET", prefix+path, nil)
	rec := httptest.NewRecorder()

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

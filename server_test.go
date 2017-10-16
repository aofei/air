package air

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	a := New()
	s := a.Server.(*server)

	assert.NotNil(t, s.air)
	assert.Equal(t, 0, len(s.pregases))
	assert.Equal(t, 0, len(s.gases))
	assert.NotNil(t, s.server)
	assert.NotNil(t, s.contextPool)
	assert.NotNil(t, s.httpErrorHandler)
}

func TestServerPrecontain(t *testing.T) {
	a := New()
	s := a.Server.(*server)
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	pregas := WrapGas(func(c *Context) error {
		return c.String("pregas")
	})

	a.Precontain(pregas)
	s.ServeHTTP(rec, req)
	assert.Equal(t, "pregas", rec.Body.String())
}

func TestServerContain(t *testing.T) {
	a := New()
	s := a.Server.(*server)
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	gas := WrapGas(func(c *Context) error {
		return c.String("gas")
	})

	a.Contain(gas)
	s.ServeHTTP(rec, req)
	assert.Equal(t, "gas", rec.Body.String())
}

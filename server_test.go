package air

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerMethodAllowed(t *testing.T) {
	a := New()
	a.server = newServer(a)

	for _, m := range methods {
		assert.True(t, methodAllowed(m))
	}

	req, _ := http.NewRequest("INVALID", "/", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

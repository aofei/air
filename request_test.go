package air

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	r := &Request{
		parseParamsOnce: &sync.Once{},
	}
	r.SetHTTPRequest(httptest.NewRequest(http.MethodGet, "/?Foo=Bar", nil))

	var s struct {
		Foo string
	}

	assert.NoError(t, r.Bind(&s))
	assert.Equal(t, "Bar", s.Foo)
}

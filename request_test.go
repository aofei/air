package air

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	r := &Request{
		Method:          "GET",
		request:         httptest.NewRequest("GET", "/?Foo=Bar", nil),
		parseParamsOnce: &sync.Once{},
	}

	var s struct {
		Foo string
	}

	assert.NoError(t, r.Bind(&s))
	assert.Equal(t, "Bar", s.Foo)
}

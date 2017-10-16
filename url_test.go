package air

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLQueryValue(t *testing.T) {
	a := New()
	c := NewContext(a)

	vs := url.Values{}
	vs.Set("name", "Air")
	vs.Set("author", "Aofei Sheng")
	req, _ := http.NewRequest("GET", "/?"+vs.Encode(), nil)

	c.feed(req, nil)

	assert.Equal(t, "Air", c.Request.URL.QueryValue("name"))
	assert.Equal(t, "Aofei Sheng", c.Request.URL.QueryValue("author"))
	assert.Equal(t, vs, c.Request.URL.QueryValues())
}

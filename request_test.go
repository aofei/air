package air

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestQueryValueAndFormValue(t *testing.T) {
	a := New()
	c := NewContext(a)

	vs := make(url.Values)
	vs.Set("name", "Air")
	vs.Set("author", "Aofei Sheng")

	req, _ := http.NewRequest(POST, "/?"+vs.Encode(), strings.NewReader(vs.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationForm)

	c.feed(req, nil)

	assert.Equal(t, "Air", c.QueryValue("name"))
	assert.Equal(t, "Aofei Sheng", c.QueryValue("author"))
	assert.Equal(t, vs.Get("name"), c.QueryValues().Get("name"))

	assert.Equal(t, "Air", c.FormValue("name"))
	assert.Equal(t, "Aofei Sheng", c.FormValue("author"))
	assert.Equal(t, vs.Get("name"), c.FormValues().Get("name"))
}

package air

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestFormValue(t *testing.T) {
	a := New()
	c := NewContext(a)

	vs := make(url.Values)
	vs.Set("name", "Air")
	vs.Set("author", "Aofei Sheng")
	req, _ := http.NewRequest(POST, "/", strings.NewReader(vs.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationForm)

	c.feed(req, nil)

	assert.Equal(t, "Air", c.Request.FormValue("name"))
	assert.Equal(t, "Aofei Sheng", c.Request.FormValue("author"))
	if fvs, err := c.Request.FormValues(); assert.NoError(t, err) {
		assert.Equal(t, vs, fvs)
	}
}

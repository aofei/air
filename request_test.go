package air

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestBind(t *testing.T) {
	a := New()
	c := NewContext(a)

	vs := make(url.Values)

	vs.Set("name", "Air")
	vs.Set("author", "Aofei Sheng")

	vs.Set("int", "1")
	vs.Set("int8", "1")
	vs.Set("int16", "1")
	vs.Set("int32", "1")
	vs.Set("int64", "1")

	vs.Set("uint", "1")
	vs.Set("uint8", "1")
	vs.Set("uint16", "1")
	vs.Set("uint32", "1")
	vs.Set("uint64", "1")

	vs.Set("bool", "true")

	vs.Set("float32", "1.11")
	vs.Set("float64", "1.11")

	type Info struct {
		Name   string `query:"name" form:"name" json:"name" xml:"name"`
		Author string `query:"author" form:"author" json:"author" xml:"author"`

		Int   int   `query:"int" form:"int" json:"int" xml:"int"`
		Int8  int8  `query:"int8" form:"int8" json:"int8" xml:"int8"`
		Int16 int16 `query:"int16" form:"int16" json:"int16" xml:"int16"`
		Int32 int32 `query:"int32" form:"int32" json:"int32" xml:"int32"`
		Int64 int64 `query:"int64" form:"int64" json:"int64" xml:"int64"`

		Uint   uint   `query:"uint" form:"uint" json:"uint" xml:"uint"`
		Uint8  uint8  `query:"uint8" form:"uint8" json:"uint8" xml:"uint8"`
		Uint16 uint16 `query:"uint16" form:"uint16" json:"uint16" xml:"uint16"`
		Uint32 uint32 `query:"uint32" form:"uint32" json:"uint32" xml:"uint32"`
		Uint64 uint64 `query:"uint64" form:"uint64" json:"uint64" xml:"uint64"`

		Bool bool `query:"bool" form:"bool" json:"bool" xml:"bool"`

		Float32 float32 `query:"float32" form:"float32" json:"float32" xml:"float32"`
		Float64 float64 `query:"float64" form:"float64" json:"float64" xml:"float64"`
	}

	raw := &Info{
		Name:   vs.Get("name"),
		Author: vs.Get("author"),

		Int:   1,
		Int8:  1,
		Int16: 1,
		Int32: 1,
		Int64: 1,

		Uint:   1,
		Uint8:  1,
		Uint16: 1,
		Uint32: 1,
		Uint64: 1,

		Bool: true,

		Float32: 1.11,
		Float64: 1.11,
	}

	j, _ := json.Marshal(raw)
	x, _ := xml.Marshal(raw)

	req, _ := http.NewRequest(GET, "/?"+vs.Encode(), nil)

	c.reset()
	c.feed(req, nil)

	i := &Info{}

	c.Bind(i)
	assert.Equal(t, *raw, *i)

	req, _ = http.NewRequest(POST, "/", strings.NewReader(vs.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationXWWWFormURLEncoded)

	c.reset()
	c.feed(req, nil)

	i = &Info{}

	c.Bind(i)
	assert.Equal(t, *raw, *i)

	req, _ = http.NewRequest(POST, "/", strings.NewReader(string(j)))
	req.Header.Add(HeaderContentType, MIMEApplicationJSON)

	c.feed(req, nil)

	i = &Info{}

	c.Bind(i)
	assert.Equal(t, *raw, *i)

	req, _ = http.NewRequest(POST, "/", strings.NewReader(string(x)))
	req.Header.Add(HeaderContentType, MIMEApplicationXML)

	c.feed(req, nil)

	i = &Info{}

	c.Bind(i)
	assert.Equal(t, *raw, *i)
}

func TestRequestFormFile(t *testing.T) {
	a := New()
	c := NewContext(a)
	req, _ := http.NewRequest(POST, "/", nil)
	req.Header.Add(HeaderContentType, MIMEMultipartFormData)
	c.feed(req, nil)
	f, fh, err := c.FormFile("air")
	assert.Nil(t, f)
	assert.Nil(t, fh)
	assert.NotNil(t, err)
}

func TestRequestCookie(t *testing.T) {
	a := New()
	c := NewContext(a)

	cookie := &http.Cookie{
		Name:  "Air",
		Value: "Aofei Sheng",
	}

	req, _ := http.NewRequest(GET, "/", nil)
	req.Header.Add("Cookie", cookie.String())

	c.feed(req, nil)

	if nc, err := c.Cookie("Air"); assert.NoError(t, err) {
		assert.Equal(t, *cookie, *nc)
	}

	assert.Contains(t, c.Cookies(), cookie)
}

func TestRequestOthers(t *testing.T) {
	a := New()
	c := NewContext(a)

	vs := make(url.Values)
	vs.Set("name", "Air")
	vs.Set("author", "Aofei Sheng")

	req, _ := http.NewRequest(POST, "/?"+vs.Encode(), strings.NewReader(vs.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationXWWWFormURLEncoded)

	c.feed(req, nil)

	assert.True(t, c.HasQueryValue("name"))
	assert.Equal(t, vs.Get("name"), c.QueryValues().Get("name"))
	assert.Equal(t, "Air", c.QueryValue("name"))
	assert.Equal(t, "Aofei Sheng", c.QueryValue("author"))

	assert.True(t, c.HasFormValue("name"))
	assert.Equal(t, vs.Get("name"), c.FormValues().Get("name"))
	assert.Equal(t, "Air", c.FormValue("name"))
	assert.Equal(t, "Aofei Sheng", c.FormValue("author"))
}

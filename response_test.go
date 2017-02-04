package air

import (
	"encoding/xml"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseRender(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	r := newRenderer(a)
	r.templates = template.Must(template.New("info").Parse("{{.name}} by {{.author}}."))
	a.Renderer = r

	c.Response.Data["name"] = "Air"
	c.Response.Data["author"] = "Aofei Sheng"
	if err := c.Response.Render("info"); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMETextHTML, rec.Header().Get(HeaderContentType))
		assert.Equal(t, "Air by Aofei Sheng.", rec.Body.String())
	}
}

func TestResponseHTML(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	if err := c.Response.HTML("Air"); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMETextHTML, rec.Header().Get(HeaderContentType))
		assert.Equal(t, "Air", rec.Body.String())
	}
}

func TestResponseString(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	if err := c.Response.String("Air"); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMETextPlain, rec.Header().Get(HeaderContentType))
		assert.Equal(t, "Air", rec.Body.String())
	}
}

func TestResponseJSON(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	info := struct{ Name, Author string }{"Air", "Aofei Sheng"}
	infoStr := `{"Name":"Air","Author":"Aofei Sheng"}`
	if err := c.Response.JSON(info); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationJSON, rec.Header().Get(HeaderContentType))
		assert.Equal(t, infoStr, rec.Body.String())
	}
}

func TestResponseJSONP(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	info := struct{ Name, Author string }{"Air", "Aofei Sheng"}
	infoStr := `{"Name":"Air","Author":"Aofei Sheng"}`
	cb := "callback"
	if err := c.Response.JSONP(info, cb); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationJavaScript, rec.Header().Get(HeaderContentType))
		assert.Equal(t, cb+"("+infoStr+");", rec.Body.String())
	}
}

func TestResponseXML(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	type Info struct{ Name, Author string }
	info := Info{"Air", "Aofei Sheng"}
	infoStr := "<Info><Name>Air</Name><Author>Aofei Sheng</Author></Info>"
	if err := c.Response.XML(info); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationXML, rec.Header().Get(HeaderContentType))
		assert.Equal(t, xml.Header+infoStr, rec.Body.String())
	}
}

func TestResponseYAML(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	info := struct{ Name, Author string }{"Air", "Aofei Sheng"}
	infoStr := "name: Air\nauthor: Aofei Sheng\n"
	if err := c.Response.YAML(info); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationYAML, rec.Header().Get(HeaderContentType))
		assert.Equal(t, infoStr, rec.Body.String())
	}
}

func TestResponseBlob(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	ct := "contentType"
	b := []byte("blob")
	if err := c.Response.Blob(ct, b); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, ct, rec.Header().Get(HeaderContentType))
		assert.Equal(t, b, rec.Body.Bytes())
	}
}

func TestResponseStream(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	s := "response from a stream"
	if err := c.Response.Stream(MIMEOctetStream, strings.NewReader(s)); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEOctetStream, rec.Header().Get(HeaderContentType))
		assert.Equal(t, s, rec.Body.String())
	}
}

func TestResponseFile(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	f := "air.go"
	b, _ := ioutil.ReadFile(f)
	if err := c.Response.File(f); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMETextPlain, rec.Header().Get(HeaderContentType))
		assert.Equal(t, b, rec.Body.Bytes())
	}
}

func TestResponseAttachment(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	f := "air.go"
	h := "attachment; filename=" + f
	b, _ := ioutil.ReadFile(f)
	if err := c.Response.Attachment(f, f); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, h, rec.Header().Get(HeaderContentDisposition))
		assert.Equal(t, b, rec.Body.Bytes())
	}
}

func TestResponseInline(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	f := "air.go"
	h := "inline; filename=" + f
	b, _ := ioutil.ReadFile(f)
	if err := c.Response.Inline(f, f); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, h, rec.Header().Get(HeaderContentDisposition))
		assert.Equal(t, b, rec.Body.Bytes())
	}
}

func TestResponseNoContent(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	c.Response.NoContent()
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", rec.Header().Get(HeaderContentDisposition))
	assert.Equal(t, "", rec.Body.String())
}

func TestResponseRedirect(t *testing.T) {
	a := New()
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	url := "https://github.com/sheng/air"
	if err := c.Response.Redirect(http.StatusMovedPermanently, url); assert.NoError(t, err) {
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
		assert.Equal(t, url, rec.Header().Get(HeaderLocation))
		assert.Equal(t, "", rec.Body.String())
	}
}

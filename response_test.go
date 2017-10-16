package air

import (
	"encoding/xml"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseSetCookie(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	cookie := &http.Cookie{
		Name:  "Air",
		Value: "An ideal RESTful web framework for Go.",
	}

	c.SetCookie(cookie)

	assert.Equal(
		t,
		cookie.String(),
		c.Response.Header().Get("Set-Cookie"),
	)
}

func TestRequestPush(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)
	c.feed(req, rec)
	assert.Panics(t, func() { c.Push("air.go", nil) })
}

func TestResponseRender(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	r := newRenderer(a)
	r.template = template.Must(
		template.New("info").Parse("{{.name}} by {{.author}}."),
	)
	a.Renderer = r

	c.Data["name"] = "Air"
	c.Data["author"] = "Aofei Sheng"
	if err := c.Render("info"); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"text/html; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, "Air by Aofei Sheng.", rec.Body.String())
	}

	c.reset()

	assert.Error(t, c.Render("unknown"))
}

func TestResponseHTML(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	if err := c.HTML("Air"); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"text/html; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, "Air", rec.Body.String())
	}
}

func TestResponseString(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	if err := c.String("Air"); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"text/plain; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, "Air", rec.Body.String())
	}
}

func TestResponseJSON(t *testing.T) {
	a := New()
	a.Config.DebugMode = true
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	info := struct{ Name, Author string }{"Air", "Aofei Sheng"}
	infoStr := `{
	"Name": "Air",
	"Author": "Aofei Sheng"
}`

	if err := c.JSON(info); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"application/json; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, infoStr, rec.Body.String())
	}

	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()

	c.reset()
	c.feed(req, rec)

	assert.Error(t, c.JSON(Air{}))
}

func TestResponseJSONP(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	info := struct{ Name, Author string }{"Air", "Aofei Sheng"}
	infoStr := `{"Name":"Air","Author":"Aofei Sheng"}`
	cb := "callback"
	if err := c.JSONP(info, cb); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"application/javascript; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, cb+"("+infoStr+");", rec.Body.String())
	}

	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()

	c.reset()
	c.feed(req, rec)

	assert.Error(t, c.JSONP(Air{}, cb))
}

func TestResponseXML(t *testing.T) {
	a := New()
	a.Config.DebugMode = true
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	type Info struct{ Name, Author string }
	info := Info{"Air", "Aofei Sheng"}
	infoStr := xml.Header + `<Info>
	<Name>Air</Name>
	<Author>Aofei Sheng</Author>
</Info>`

	if err := c.XML(info); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"application/xml; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, infoStr, rec.Body.String())
	}

	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()

	c.reset()
	c.feed(req, rec)

	assert.Error(t, c.XML(Air{}))
}

func TestResponseBlob(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	ct := "contentType"
	b := []byte("blob")
	if err := c.Blob(ct, b); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, ct, rec.Header().Get("Content-Type"))
		assert.Equal(t, b, rec.Body.Bytes())
	}

	c.Response.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode)
	assert.Equal(t, len(b), c.Response.Size)
}

func TestResponseStream(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	s := "response from a stream"
	if assert.NoError(
		t,
		c.Stream("application/javascript", strings.NewReader(s)),
	) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"application/javascript",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, s, rec.Body.String())
	}
}

func TestResponseFile(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	f := "air.go"
	b, _ := ioutil.ReadFile(f)
	if err := c.File(f); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"text/plain; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, b, rec.Body.Bytes())
	}

	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()

	c.reset()
	c.feed(req, rec)

	assert.True(t, os.IsNotExist(c.File("file_not_exist.html")))

	file, _ := os.Create("index.html")
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()
	file.WriteString("<html></html>")

	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()

	c.reset()
	c.feed(req, rec)

	if err := c.File("."); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"text/html; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, "<html></html>", rec.Body.String())
	}

	a.Config.CofferEnabled = true
	a.Config.AssetRoot = "."
	a.Coffer.Init()

	req, _ = http.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()

	c.reset()
	c.feed(req, rec)

	if err := c.File("."); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(
			t,
			"text/html; charset=utf-8",
			rec.Header().Get("Content-Type"),
		)
		assert.Equal(t, "<html></html>", rec.Body.String())
	}
}

func TestResponseAttachment(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	f := "air.go"
	h := "attachment; filename=" + f
	b, _ := ioutil.ReadFile(f)
	if err := c.Attachment(f, f); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, h, rec.Header().Get("Content-Disposition"))
		assert.Equal(t, b, rec.Body.Bytes())
	}
}

func TestResponseInline(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	f := "air.go"
	h := "inline; filename=" + f
	b, _ := ioutil.ReadFile(f)
	if err := c.Inline(f, f); assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, h, rec.Header().Get("Content-Disposition"))
		assert.Equal(t, b, rec.Body.Bytes())
	}
}

func TestResponseNoContent(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	c.NoContent(http.StatusOK)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", rec.Header().Get("Content-Disposition"))
	assert.Equal(t, "", rec.Body.String())
}

func TestResponseRedirect(t *testing.T) {
	a := New()
	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(a)

	c.feed(req, rec)

	url := "https://github.com/sheng/air"
	err := c.Redirect(http.StatusMovedPermanently, url)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
		assert.Equal(t, url, rec.Header().Get("Location"))
		assert.Equal(t, "", rec.Body.String())
	}
}

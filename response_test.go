package air

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com/foo/bar?foo=bar#foobar",
		bytes.NewBufferString("foobar"),
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.Equal(t, req, r.request)
	assert.Equal(t, rec, r.writer)
	assert.Equal(t, rec, r.flusher)
	assert.Nil(t, r.hijacker)
	assert.Nil(t, r.closeNotifier)
	assert.Nil(t, r.pusher)
	assert.Equal(t, 200, r.StatusCode)
	assert.NotNil(t, r.Headers)
	assert.Zero(t, len(r.Cookies))
	assert.Zero(t, r.Size)
	assert.False(t, r.Written)
}

func TestResponseWrite(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	r.Headers["Content-Type"] = "text/html"
	r.Cookies = append(r.Cookies, &Cookie{
		Name:     "foo",
		Value:    "bar",
		Expires:  time.Now().Add(time.Hour),
		MaxAge:   3600,
		Domain:   "example.com",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
	})

	html := `<!DOCTYPE html>
<html>
<head>
<title>foobar</title>
</head>
<body>
<h1>foobar</h1>
</body>
</html>`

	assert.Nil(t, r.write([]byte(html)))
	assert.Equal(t, r.StatusCode, rec.Code)
	assert.Equal(t, len(html), r.Size)
	assert.True(t, r.Written)
	assert.Equal(
		t,
		r.Headers["Content-Type"],
		rec.Header().Get("Content-Type"),
	)
	assert.Equal(t, r.Cookies[0].String(), rec.Header().Get("Set-Cookie"))
	assert.Equal(t, html, rec.Body.String())
}

func TestResponseNoContent(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.Nil(t, r.NoContent())
	assert.Equal(t, 0, r.Size)
}

func TestResponseRedirect(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)

	url := "https://example.com/foobar"

	assert.Nil(t, r.Redirect(url))
	assert.Equal(t, 0, r.Size)
	assert.Equal(t, url, rec.Header().Get("Location"))
}

func TestResponseBlob(t *testing.T) {
	a := New()
	a.MinifierEnabled = true
	assert.Nil(t, a.minifier.init())
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.Nil(t, r.Blob("text/html", []byte("<!DOCTYPE html>")))
	assert.Equal(
		t,
		r.Headers["Content-Type"],
		rec.Header().Get("Content-Type"),
	)
	assert.Equal(t, "<!doctype html>", rec.Body.String())
}

func TestResponseString(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.Nil(t, r.String("foobar"))
	assert.Equal(
		t,
		r.Headers["Content-Type"],
		rec.Header().Get("Content-Type"),
	)
	assert.Equal(t, "foobar", rec.Body.String())
}

func TestResponseJSON(t *testing.T) {
	a := New()
	a.DebugMode = true
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.NotNil(t, r.JSON(Air{}))
	assert.Nil(t, r.JSON(map[string]string{
		"foo": "bar",
	}))
	assert.Equal(
		t,
		r.Headers["Content-Type"],
		rec.Header().Get("Content-Type"),
	)
	assert.Equal(t, "{\n\t\"foo\": \"bar\"\n}", rec.Body.String())
}

func TestResponseXML(t *testing.T) {
	a := New()
	a.DebugMode = true
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.NotNil(t, r.XML(Air{}))

	type Info struct {
		Foobar string
	}

	assert.Nil(t, r.XML(Info{
		Foobar: "foobar",
	}))
	assert.Equal(
		t,
		r.Headers["Content-Type"],
		rec.Header().Get("Content-Type"),
	)
	assert.Equal(
		t,
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Info>\n\t"+
			"<Foobar>foobar</Foobar>\n</Info>",
		rec.Body.String(),
	)
}

func TestResponseHTML(t *testing.T) {
	a := New()
	req := newRequest(a, httptest.NewRequest(
		"GET",
		"https://example.com",
		nil,
	))
	rec := httptest.NewRecorder()

	r := newResponse(req, rec)
	assert.Nil(t, r.HTML("<!DOCTYPE html>"))
	assert.Equal(
		t,
		r.Headers["Content-Type"],
		rec.Header().Get("Content-Type"),
	)
	assert.Equal(t, "<!DOCTYPE html>", rec.Body.String())
}

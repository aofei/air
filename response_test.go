package air

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestResponseHTTPRequest(t *testing.T) {
	a := New()

	req, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.Equal(t, a, res.Air)
	assert.Equal(t, req, res.req)
	assert.NotNil(t, res.hrw)
	assert.NotNil(t, res.ohrw)

	hrw := res.HTTPResponseWriter()
	assert.NotNil(t, hrw)
	assert.Equal(t, res.Header, hrw.Header())
}

func TestResponseSetHTTPResponseWriter(t *testing.T) {
	a := New()

	_, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	hrw := httptest.NewRecorder()

	res.SetHTTPResponseWriter(hrw)
	assert.Equal(t, hrw, res.hrw)
	assert.Equal(t, hrw.Header(), res.Header)
	assert.Equal(t, hrw, res.Body)
}

func TestResponseSetCookie(t *testing.T) {
	a := New()

	_, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	res.SetCookie(&http.Cookie{})
	assert.Empty(t, res.Header.Get("Set-Cookie"))

	res.SetCookie(&http.Cookie{
		Name:  "foo",
		Value: "bar",
	})
	assert.Equal(t, "foo=bar", res.Header.Get("Set-Cookie"))
}

func TestResponseWrite(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.NoError(t, res.Write(nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	assert.NoError(t, res.Write(strings.NewReader("foobar")))
	assert.Equal(t, "foobar", rec.Body.String())

	_, res, rec = fakeRRCycle(a, http.MethodHead, "/", nil)

	assert.NoError(t, res.Write(nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	assert.NoError(t, res.Write(strings.NewReader("foobar")))
	assert.Empty(t, rec.Body.String())

	_, res, _ = fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.Error(t, res.Write(&readErrorReader{
		Seeker: strings.NewReader("foobar"),
	}))
	assert.Error(t, res.Write(&seekErrorSeeker{
		Reader: strings.NewReader("foobar"),
	}))
	assert.NoError(t, res.Write(strings.NewReader("foobar")))
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		res.Header.Get("Content-Type"),
	)

	_, res, _ = fakeRRCycle(a, http.MethodGet, "/", nil)

	a.MinifierEnabled = true

	res.Header.Set("Content-Type", "text/html; charset=utf-8")
	assert.Error(t, res.Write(&readErrorReader{
		Seeker: strings.NewReader("<!DOCTYPE html>"),
	}))

	res.Header.Set("Content-Type", "application/json; charset=utf-8")
	assert.Error(t, res.Write(strings.NewReader("{")))

	res.Header.Set("Content-Type", "text/html; charset=utf-8")
	res.SetHTTPResponseWriter(&nopResponseWriter{
		ResponseWriter: res.HTTPResponseWriter(),
	})
	assert.NoError(t, res.Write(strings.NewReader("<!DOCTYPE html>")))

	a.MinifierEnabled = false

	req, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	req.Header.Set("Range", "bytes 1-0")
	res.Header.Set(
		"Last-Modified",
		time.Unix(0, 0).UTC().Format(http.TimeFormat),
	)

	assert.Error(t, res.Write(strings.NewReader("foobar")))

	_, res, _ = fakeRRCycle(a, http.MethodGet, "/", nil)
	res.Status = http.StatusInternalServerError
	res.Header.Set("Content-Type", "text/plain; charset=utf-8")

	assert.Error(t, res.Write(&seekEndErrorSeeker{
		Reader: strings.NewReader("foobar"),
	}))
	assert.Error(t, res.Write(&seekStartErrorSeeker{
		Reader: strings.NewReader("foobar"),
	}))
	assert.NoError(t, res.Write(strings.NewReader("foobar")))

	_, res, _ = fakeRRCycle(a, http.MethodHead, "/", nil)
	res.Status = http.StatusInternalServerError
	res.Header.Set("Content-Type", "text/plain; charset=utf-8")

	assert.NoError(t, res.Write(strings.NewReader("foobar")))
}

func TestResponseWriteString(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.NoError(t, res.WriteString("foobar"))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "foobar", rec.Body.String())
}

func TestResponseWriteJSON(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	var foobar struct {
		Foo string `json:"foo"`
	}
	foobar.Foo = "bar"

	assert.Error(t, res.WriteJSON(&errorJSONMarshaler{}))
	assert.NoError(t, res.WriteJSON(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/json; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, `{"foo":"bar"}`, rec.Body.String())

	_, res, rec = fakeRRCycle(a, http.MethodGet, "/", nil)

	a.DebugMode = true

	assert.Error(t, res.WriteJSON(&errorJSONMarshaler{}))
	assert.NoError(t, res.WriteJSON(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/json; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "{\n\t\"foo\": \"bar\"\n}", rec.Body.String())
}

func TestResponseWriteXML(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	var foobar struct {
		XMLName xml.Name `xml:"foobar"`
		Foo     string   `xml:"foo"`
	}
	foobar.Foo = "bar"

	assert.Error(t, res.WriteXML(&errorXMLMarshaler{}))
	assert.NoError(t, res.WriteXML(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/xml; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(
		t,
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<foobar><foo>bar</foo></foobar>",
		rec.Body.String(),
	)

	_, res, rec = fakeRRCycle(a, http.MethodGet, "/", nil)

	a.DebugMode = true

	assert.Error(t, res.WriteXML(&errorXMLMarshaler{}))
	assert.NoError(t, res.WriteXML(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/xml; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(
		t,
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<foobar>\n\t<foo>bar</foo>\n</foobar>",
		rec.Body.String(),
	)
}

func TestResponseWriteProtobuf(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.NoError(t, res.WriteProtobuf(&wrapperspb.StringValue{
		Value: "foobar",
	}))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/protobuf",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "\n\x06foobar", rec.Body.String())
}

func TestResponseWriteMsgpack(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	var foobar struct {
		Foo string `msgpack:"foo"`
	}
	foobar.Foo = "bar"

	assert.Error(t, res.WriteMsgpack(&errorMsgpackMarshaler{}))
	assert.NoError(t, res.WriteMsgpack(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/msgpack",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "\x81\xa3foo\xa3bar", rec.Body.String())
}

func TestResponseWriteTOML(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	var foobar struct {
		Foo string `toml:"foo"`
	}
	foobar.Foo = "bar"

	assert.Error(t, res.WriteTOML(""))
	assert.NoError(t, res.WriteTOML(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/toml; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "foo = \"bar\"\n", rec.Body.String())
}

func TestResponseWriteYAML(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	var foobar struct {
		Foo string `yaml:"foo"`
	}
	foobar.Foo = "bar"

	assert.Error(t, res.WriteYAML(&errorYAMLMarshaler{}))
	assert.NoError(t, res.WriteYAML(&foobar))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"application/yaml; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "foo: bar\n", rec.Body.String())
}

func TestResponseWriteHTML(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.NoError(t, res.WriteHTML("<!DOCTYPE html>"))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"text/html; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "<!DOCTYPE html>", rec.Body.String())

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	a.AutoPushEnabled = true

	hr := req.HTTPRequest()
	hr.ProtoMajor = 2

	req.SetHTTPRequest(hr)

	assert.NoError(t, res.WriteHTML(`
<!DOCTYPE html>
<html>
	<head>
		<link rel="stylesheet" href="/assets/css/main.css">
		<link rel="preload" href="/assets/css/style.css" as="style">
		<link href="/assets/css/theme.css">
		<link rel="shortcut icon" href="/favicon.ico">
	</head>

	<body>
		<img src="/assets/images/avatar.jpg">
		<script src="/assets/js/main.js"></script>
	</body>
</html>
	`))
}

func TestResponseRender(t *testing.T) {
	a := New()

	dir, err := ioutil.TempDir("", "air.TestResponseRender")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.RendererTemplateRoot = dir

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(a.RendererTemplateRoot, "test.html"),
		[]byte(`<a href="/">Go Home</a>`),
		os.ModePerm,
	))

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.Error(t, res.Render(nil, "foobar.html"))
	assert.NoError(t, res.Render(nil, "test.html"))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"text/html; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, `<a href="/">Go Home</a>`, rec.Body.String())
}

func TestResponseRedirect(t *testing.T) {
	a := New()

	_, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.NoError(t, res.Redirect("http://example.com/foo/bar"))
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(
		t,
		"http://example.com/foo/bar",
		rec.HeaderMap.Get("Location"),
	)
}

func TestResponseDefer(t *testing.T) {
	a := New()

	_, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	res.Defer(nil)
	assert.Len(t, res.deferredFuncs, 0)

	res.Defer(func() {})
	assert.Len(t, res.deferredFuncs, 1)
}

func TestNewReverseProxyBufferPool(t *testing.T) {
	rpbp := newReverseProxyBufferPool()

	assert.NotNil(t, rpbp.pool)
}

func TestReverseProxyBufferPoolGet(t *testing.T) {
	rpbp := newReverseProxyBufferPool()

	assert.Len(t, rpbp.Get(), 32<<20)
}

func TestReverseProxyBufferPoolPut(t *testing.T) {
	rpbp := newReverseProxyBufferPool()

	rpbp.Put(make([]byte, 32<<20))
}

type nopResponseWriter struct {
	http.ResponseWriter
}

func (nrw *nopResponseWriter) WriteHeader(int) {
}

func (nrw *nopResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

type readErrorReader struct {
	io.Seeker
}

func (rer *readErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

type seekErrorSeeker struct {
	io.Reader
}

func (ses *seekErrorSeeker) Seek(int64, int) (int64, error) {
	return 0, errors.New("seek error")
}

type seekStartErrorSeeker struct {
	io.Reader
}

func (sses *seekStartErrorSeeker) Seek(_ int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		return 0, errors.New("seek start error")
	}

	return 0, nil
}

type seekEndErrorSeeker struct {
	io.Reader
}

func (sees *seekEndErrorSeeker) Seek(_ int64, whence int) (int64, error) {
	if whence == io.SeekEnd {
		return 0, errors.New("seek end error")
	}

	return 0, nil
}

type errorJSONMarshaler struct {
}

func (ejm *errorJSONMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal json error")
}

type errorXMLMarshaler struct {
}

func (exm *errorXMLMarshaler) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return errors.New("marshal xml error")
}

type errorMsgpackMarshaler struct {
}

func (emm *errorMsgpackMarshaler) MarshalMsgpack() ([]byte, error) {
	return nil, errors.New("marshal msgpack error")
}

type errorYAMLMarshaler struct {
}

func (eym *errorYAMLMarshaler) MarshalYAML() (interface{}, error) {
	return nil, errors.New("marshal yaml error")
}

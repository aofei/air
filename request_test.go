package air

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestHTTPRequest(t *testing.T) {
	a := New()

	req, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.Equal(t, a, req.Air)
	assert.Equal(t, res, req.res)
	assert.Len(t, req.params, 0)
	assert.Len(t, req.routeParamNames, 0)
	assert.Len(t, req.routeParamValues, 0)
	assert.NotNil(t, req.parseRouteParamsOnce)
	assert.NotNil(t, req.parseOtherParamsOnce)
	assert.Nil(t, req.localizedString)

	hr := req.HTTPRequest()
	assert.NotNil(t, hr)
	assert.Equal(t, req.Method, hr.Method)
	assert.Equal(t, req.Authority, hr.Host)
	assert.Equal(t, req.Path, hr.RequestURI)
	assert.Equal(t, req.Header, hr.Header)
	assert.Equal(t, req.Body, hr.Body)
	assert.Equal(t, req.ContentLength, hr.ContentLength)
	assert.Equal(t, req.Context, hr.Context())

	req.Path = "/foobar?foo=bar"
	req.Body = &bytes.Buffer{}
	req.Context = context.WithValue(req.Context, "foo", "bar")

	hr = req.HTTPRequest()
	assert.Equal(t, req.Path, hr.RequestURI)
	assert.NotEqual(t, req.Body, hr.Body)
	assert.Equal(t, req.Context, hr.Context())
}

func TestRequestSetHTTPRequest(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	hr := httptest.NewRequest(
		http.MethodPost,
		"https://example.com/foobar?foo=bar",
		nil,
	)
	hr.Trailer = http.Header{
		"Foo": []string{},
	}

	req.SetHTTPRequest(hr)
	assert.Equal(t, hr, req.hr)
	assert.Equal(t, hr.Method, req.Method)
	assert.Equal(t, "https", req.Scheme)
	assert.Equal(t, hr.Host, req.Authority)
	assert.Equal(t, hr.RequestURI, req.Path)
	assert.Equal(t, hr.Header, req.Header)
	assert.Equal(t, "Foo", req.Header.Get("Trailer"))
	assert.Equal(t, hr.Body, req.Body)
	assert.Equal(t, hr.ContentLength, req.ContentLength)
	assert.Equal(t, hr.Context(), req.Context)
}

func TestRequestRemoteAddress(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.Equal(t, "192.0.2.1:1234", req.RemoteAddress())
	assert.Equal(t, "192.0.2.1:1234", req.ClientAddress())

	req.Header.Set("X-Forwarded-For", "192.0.2.2:1234, 192.0.2.3:1234")
	assert.Equal(t, "192.0.2.2:1234", req.ClientAddress())

	req.Header.Set("Forwarded", "for=192.0.2.4:1234, for=192.0.2.5:1234")
	assert.Equal(t, "192.0.2.4:1234", req.ClientAddress())

	req.Header.Set("Forwarded", `for="[2001:db8:cafe::17]"`)
	assert.Equal(t, "[2001:db8:cafe::17]", req.ClientAddress())

	req.Header.Set("Forwarded", `for="[2001:db8:cafe::17]:4711"`)
	assert.Equal(t, "[2001:db8:cafe::17]:4711", req.ClientAddress())

	req.Header.Set("Forwarded", `FoR="[2001:Db8:CaFe::17]"`)
	assert.Equal(t, "[2001:Db8:CaFe::17]", req.ClientAddress())
}

func TestRequestCookie(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	hr := req.HTTPRequest()
	hr.AddCookie(&http.Cookie{
		Name:  "foo",
		Value: "bar",
	})

	c := req.Cookie("foo")
	assert.NotNil(t, c)
	assert.Equal(t, "bar", c.Value)

	c = req.Cookie("bar")
	assert.Nil(t, c)
}

func TestRequestCookies(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	cs := req.Cookies()
	assert.NotNil(t, cs)
	assert.Len(t, cs, 0)

	hr := req.HTTPRequest()
	hr.AddCookie(&http.Cookie{
		Name:  "foo",
		Value: "bar",
	})
	hr.AddCookie(&http.Cookie{
		Name:  "bar",
		Value: "foo",
	})

	cs = req.Cookies()
	assert.NotNil(t, cs)
	assert.Len(t, cs, 2)
	assert.Equal(t, "foo", cs[0].Name)
	assert.Equal(t, "bar", cs[0].Value)
	assert.Equal(t, "bar", cs[1].Name)
	assert.Equal(t, "foo", cs[1].Value)
}

func TestRequestParam(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	p := req.Param("foo")
	assert.Nil(t, p)

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/?foo=bar", nil)

	p = req.Param("foo")
	assert.NotNil(t, p)
	assert.Len(t, p.Values, 1)
	assert.Equal(t, "bar", p.Value().String())

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/?foo=bar&foo=bar2", nil)

	p = req.Param("foo")
	assert.NotNil(t, p)
	assert.Len(t, p.Values, 2)
	assert.Equal(t, "bar", p.Values[0].String())
	assert.Equal(t, "bar2", p.Values[1].String())

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/?foo=bar1&foo=bar2", nil)
	req.routeParamNames = []string{"foo"}
	req.routeParamValues = []string{"bar"}

	p = req.Param("foo")
	assert.NotNil(t, p)
	assert.Len(t, p.Values, 3)
	assert.Equal(t, "bar", p.Values[0].String())
	assert.Equal(t, "bar1", p.Values[1].String())
	assert.Equal(t, "bar2", p.Values[2].String())
}

func TestRequestParams(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/?foo=bar&bar=foo", nil)
	req.routeParamNames = []string{"Foo"}
	req.routeParamValues = []string{"bar"}

	ps := req.Params()
	assert.Len(t, ps, 3)
}

func TestRequestParseRouteParams(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/?foo=bar2", nil)
	req.routeParamNames = []string{"foo", "bar"}
	req.routeParamValues = []string{"bar", "foo%air"}

	req.parseOtherParamsOnce.Do(req.parseOtherParams)
	req.parseRouteParamsOnce.Do(req.parseRouteParams)

	assert.Nil(t, req.routeParamNames)
	assert.Nil(t, req.routeParamValues)

	assert.Len(t, req.Params(), 2)
	assert.Len(t, req.Param("foo").Values, 2)
	assert.Equal(t, "bar", req.Param("foo").Values[0].String())
	assert.Equal(t, "bar2", req.Param("foo").Values[1].String())
	assert.Len(t, req.Param("bar").Values, 1)
	assert.Equal(t, "foo%air", req.Param("bar").Values[0].String())
}

func TestRequestParseOtherParams(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/?foo=bar2&bar=foo", nil)
	req.routeParamNames = []string{"foo"}
	req.routeParamValues = []string{"bar"}

	req.parseRouteParamsOnce.Do(req.parseRouteParams)
	req.parseOtherParamsOnce.Do(req.parseOtherParams)

	assert.Len(t, req.Params(), 2)
	assert.Len(t, req.Param("foo").Values, 2)
	assert.Equal(t, "bar", req.Param("foo").Values[0].String())
	assert.Equal(t, "bar2", req.Param("foo").Values[1].String())
	assert.Len(t, req.Param("bar").Values, 1)
	assert.Equal(t, "foo", req.Param("bar").Values[0].String())

	req, _, _ = fakeRRCycle(a, http.MethodGet, "/?foo=bar", nil)

	hr := req.HTTPRequest()
	hr.Form = url.Values{
		"bar": []string{},
	}

	req.parseRouteParamsOnce.Do(req.parseRouteParams)
	req.parseOtherParamsOnce.Do(req.parseOtherParams)

	assert.Len(t, req.Params(), 0)

	buf := bytes.Buffer{}
	writer := multipart.NewWriter(&buf)
	writer.WriteField("foo", "bar2")
	writer.WriteField("foobar", "barfoo")

	w, err := writer.CreateFormFile("bar", "foo.bar")
	assert.NoError(t, err)
	assert.NotNil(t, w)

	n, err := w.Write([]byte("foobar"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)

	w, err = writer.CreateFormFile("barfoo", "barfoo.foobar")
	assert.NoError(t, err)
	assert.NotNil(t, w)

	n, err = w.Write([]byte("barfoo"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)

	assert.NoError(t, writer.Close())

	req, _, _ = fakeRRCycle(a, http.MethodPost, "/", &buf)
	req.routeParamNames = []string{"foo", "bar"}
	req.routeParamValues = []string{"bar", "foo"}

	hr = req.HTTPRequest()
	hr.Header.Set("Content-Type", writer.FormDataContentType())

	req.parseRouteParamsOnce.Do(req.parseRouteParams)
	req.parseOtherParamsOnce.Do(req.parseOtherParams)

	assert.Len(t, req.Params(), 4)

	buf.Reset()
	assert.NoError(t, multipart.NewWriter(&buf).Close())

	req, _, _ = fakeRRCycle(a, http.MethodPost, "/", &buf)

	hr = req.HTTPRequest()
	hr.Header.Set("Content-Type", writer.FormDataContentType())

	hr = req.HTTPRequest()
	hr.MultipartForm = &multipart.Form{
		Value: map[string][]string{
			"foo": {},
		},
		File: map[string][]*multipart.FileHeader{
			"boo": {},
		},
	}

	req.parseRouteParamsOnce.Do(req.parseRouteParams)
	req.parseOtherParamsOnce.Do(req.parseOtherParams)

	assert.Len(t, req.Params(), 0)
}

func TestRequestGrowParams(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	req.growParams(0)
	assert.Len(t, req.params, 0)
	assert.Equal(t, 0, cap(req.params))

	req.growParams(1)
	assert.Len(t, req.params, 0)
	assert.Equal(t, 1, cap(req.params))
}

func TestRequestBind(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/",
		strings.NewReader(`{"foo":"bar"}`),
	)

	var foobar struct {
		Foo string `json:"foo"`
	}

	assert.Error(t, req.Bind(&foobar))

	req.Header.Set("Content-Type", "application/json")
	assert.NoError(t, req.Bind(&foobar))
	assert.Equal(t, "bar", foobar.Foo)
}

func TestRequestLocalizedString(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)

	assert.Equal(t, "foo", req.LocalizedString("foo"))

	a.I18nEnabled = true

	dir, err := ioutil.TempDir("", "air.TestRequestLocalizedString")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a.I18nLocaleRoot = dir

	assert.Equal(t, "foo", req.LocalizedString("foo"))
}

func TestRequestParamValue(t *testing.T) {
	rp := &RequestParam{
		Name: "foo",
	}
	assert.Nil(t, rp.Values)
	assert.Nil(t, rp.Value())

	rp.Values = []*RequestParamValue{
		{
			i: "bar",
		},
		{
			i: "foobar",
		},
	}
	assert.Equal(t, "bar", rp.Value().String())
}

func TestRequestParamValueBool(t *testing.T) {
	rpv := &RequestParamValue{
		i: "true",
	}
	assert.Nil(t, rpv.b)

	b, err := rpv.Bool()
	assert.NoError(t, err)
	assert.True(t, b)
	assert.NotNil(t, rpv.b)

	rpv = &RequestParamValue{
		i: "eslaf",
	}
	assert.Nil(t, rpv.b)

	b, err = rpv.Bool()
	assert.Error(t, err)
	assert.False(t, b)
	assert.Nil(t, rpv.b)
}

func TestRequestParamValueInt(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.i64)

	i, err := rpv.Int()
	assert.NoError(t, err)
	assert.Equal(t, 80, i)
	assert.NotNil(t, rpv.i64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.i64)

	i, err = rpv.Int()
	assert.Error(t, err)
	assert.Zero(t, i)
	assert.Nil(t, rpv.i64)
}

func TestRequestParamValueInt8(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.i64)

	i8, err := rpv.Int8()
	assert.NoError(t, err)
	assert.Equal(t, int8(80), i8)
	assert.NotNil(t, rpv.i64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.i64)

	i8, err = rpv.Int8()
	assert.Error(t, err)
	assert.Zero(t, i8)
	assert.Nil(t, rpv.i64)
}

func TestRequestParamValueInt16(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.i64)

	i16, err := rpv.Int16()
	assert.NoError(t, err)
	assert.Equal(t, int16(80), i16)
	assert.NotNil(t, rpv.i64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.i64)

	i16, err = rpv.Int16()
	assert.Error(t, err)
	assert.Zero(t, i16)
	assert.Nil(t, rpv.i64)
}

func TestRequestParamValueInt32(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.i64)

	i32, err := rpv.Int32()
	assert.NoError(t, err)
	assert.Equal(t, int32(80), i32)
	assert.NotNil(t, rpv.i64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.i64)

	i32, err = rpv.Int32()
	assert.Error(t, err)
	assert.Zero(t, i32)
	assert.Nil(t, rpv.i64)
}

func TestRequestParamValueInt64(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.i64)

	i64, err := rpv.Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(80), i64)
	assert.NotNil(t, rpv.i64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.i64)

	i64, err = rpv.Int64()
	assert.Error(t, err)
	assert.Zero(t, i64)
	assert.Nil(t, rpv.i64)
}

func TestRequestParamValueUint(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.ui64)

	ui, err := rpv.Uint()
	assert.NoError(t, err)
	assert.Equal(t, uint(80), ui)
	assert.NotNil(t, rpv.ui64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.ui64)

	ui, err = rpv.Uint()
	assert.Error(t, err)
	assert.Zero(t, ui)
	assert.Nil(t, rpv.ui64)
}

func TestRequestParamValueUint8(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.ui64)

	ui8, err := rpv.Uint8()
	assert.NoError(t, err)
	assert.Equal(t, uint8(80), ui8)
	assert.NotNil(t, rpv.ui64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.ui64)

	ui8, err = rpv.Uint8()
	assert.Error(t, err)
	assert.Zero(t, ui8)
	assert.Nil(t, rpv.ui64)
}

func TestRequestParamValueUint16(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.ui64)

	ui16, err := rpv.Uint16()
	assert.NoError(t, err)
	assert.Equal(t, uint16(80), ui16)
	assert.NotNil(t, rpv.ui64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.ui64)

	ui16, err = rpv.Uint16()
	assert.Error(t, err)
	assert.Zero(t, ui16)
	assert.Nil(t, rpv.ui64)
}

func TestRequestParamValueUint32(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.ui64)

	ui32, err := rpv.Uint32()
	assert.NoError(t, err)
	assert.Equal(t, uint32(80), ui32)
	assert.NotNil(t, rpv.ui64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.ui64)

	ui32, err = rpv.Uint32()
	assert.Error(t, err)
	assert.Zero(t, ui32)
	assert.Nil(t, rpv.ui64)
}

func TestRequestParamValueUint64(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.ui64)

	ui64, err := rpv.Uint64()
	assert.NoError(t, err)
	assert.Equal(t, uint64(80), ui64)
	assert.NotNil(t, rpv.ui64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.ui64)

	ui64, err = rpv.Uint64()
	assert.Error(t, err)
	assert.Zero(t, ui64)
	assert.Nil(t, rpv.ui64)
}

func TestRequestParamValueFloat32(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.f64)

	f32, err := rpv.Float32()
	assert.NoError(t, err)
	assert.Equal(t, float32(80), f32)
	assert.NotNil(t, rpv.f64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.f64)

	f32, err = rpv.Float32()
	assert.Error(t, err)
	assert.Zero(t, f32)
	assert.Nil(t, rpv.f64)
}

func TestRequestParamValueFloat64(t *testing.T) {
	rpv := &RequestParamValue{
		i: "80",
	}
	assert.Nil(t, rpv.f64)

	f64, err := rpv.Float64()
	assert.NoError(t, err)
	assert.Equal(t, float64(80), f64)
	assert.NotNil(t, rpv.f64)

	rpv = &RequestParamValue{
		i: "八零",
	}
	assert.Nil(t, rpv.f64)

	f64, err = rpv.Float64()
	assert.Error(t, err)
	assert.Zero(t, f64)
	assert.Nil(t, rpv.f64)
}

func TestRequestParamValueString(t *testing.T) {
	rpv := &RequestParamValue{
		i: "foobar",
	}
	assert.Nil(t, rpv.s)

	s := rpv.String()
	assert.Equal(t, "foobar", s)
	assert.NotNil(t, rpv.s)

	rpv = &RequestParamValue{
		i: errors.New("foobar"),
	}
	assert.Nil(t, rpv.s)

	s = rpv.String()
	assert.Equal(t, "foobar", s)
	assert.NotNil(t, rpv.s)
}

func TestRequestParamValueFile(t *testing.T) {
	rpv := &RequestParamValue{
		i: &multipart.FileHeader{},
	}
	assert.Nil(t, rpv.f)

	f, err := rpv.File()
	assert.NoError(t, err)
	assert.NotNil(t, f)
	assert.NotNil(t, rpv.f)

	rpv = &RequestParamValue{
		i: "foobar",
	}
	assert.Nil(t, rpv.f)

	f, err = rpv.File()
	assert.Equal(t, http.ErrMissingFile, err)
	assert.Nil(t, f)
	assert.Nil(t, rpv.f)
}

func TestRequestBodyRead(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	hr := req.HTTPRequest()

	rb := &requestBody{
		r:  req,
		hr: hr,
		rc: hr.Body,
	}
	hr.Body = rb
	req.SetHTTPRequest(hr)

	n, err := rb.Read(nil)
	assert.Equal(t, io.EOF, err)
	assert.Zero(t, n)

	n, err = rb.Read(nil)
	assert.Equal(t, io.EOF, err)
	assert.Zero(t, n)

	req, _, _ = fakeRRCycle(
		a,
		http.MethodGet,
		"/",
		strings.NewReader("foobar"),
	)
	hr = req.HTTPRequest()

	rb = &requestBody{
		r:  req,
		hr: hr,
		rc: hr.Body,
	}
	hr.Body = rb
	req.SetHTTPRequest(hr)

	n, err = rb.Read(nil)
	assert.NoError(t, err)
	assert.Zero(t, n)

	b := make([]byte, 3)
	n, err = rb.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "foo", string(b))

	b = make([]byte, 4)
	n, err = rb.Read(b)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "bar\x00", string(b))

	req, _, _ = fakeRRCycle(
		a,
		http.MethodGet,
		"/",
		strings.NewReader("foobar"),
	)
	hr = req.HTTPRequest()
	hr.ContentLength = -1

	rb = &requestBody{
		r:  req,
		hr: hr,
		rc: hr.Body,
	}
	hr.Body = rb
	req.SetHTTPRequest(hr)

	n, err = rb.Read(nil)
	assert.NoError(t, err)
	assert.Zero(t, n)

	b = make([]byte, 3)
	n, err = rb.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "foo", string(b))

	b = make([]byte, 3)
	n, err = rb.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "bar", string(b))

	b = make([]byte, 1)
	n, err = rb.Read(b)
	assert.Equal(t, io.EOF, err)
	assert.Zero(t, n)

	req, _, _ = fakeRRCycle(
		a,
		http.MethodGet,
		"/",
		strings.NewReader("foobar"),
	)
	hr = req.HTTPRequest()
	hr.ContentLength = 3

	rb = &requestBody{
		r:  req,
		hr: hr,
		rc: &eofCloser{
			Reader: hr.Body,
		},
	}
	hr.Body = rb
	req.SetHTTPRequest(hr)

	n, err = rb.Read(nil)
	assert.NoError(t, err)
	assert.Zero(t, n)

	b = make([]byte, 3)
	n, err = rb.Read(b)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "foo", string(b))

	req, _, _ = fakeRRCycle(
		a,
		http.MethodGet,
		"/",
		strings.NewReader("foobar"),
	)
	hr = req.HTTPRequest()
	hr.Trailer = http.Header{
		"Foo": []string{},
	}

	rb = &requestBody{
		r:  req,
		hr: hr,
		rc: hr.Body,
	}
	hr.Body = rb
	req.SetHTTPRequest(hr)

	b = make([]byte, 6)
	n, err = rb.Read(b)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "foobar", string(b))
}

func TestRequestBodyClose(t *testing.T) {
	a := New()

	req, _, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	hr := req.HTTPRequest()

	rb := &requestBody{
		r:  req,
		hr: hr,
		rc: hr.Body,
	}
	hr.Body = rb
	req.SetHTTPRequest(hr)

	assert.NoError(t, rb.Close())
}

type eofCloser struct {
	io.Reader
}

func (ec *eofCloser) Close() error {
	return io.EOF
}

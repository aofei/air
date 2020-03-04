package air

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNewBinder(t *testing.T) {
	a := New()
	b := a.binder

	assert.NotNil(t, b)
	assert.NotNil(t, b.a)
}

func TestBindGETNoBody(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Bool    bool    `param:"bool"`
		Int     int     `param:"int"`
		Int8    int8    `param:"int8"`
		Int16   int16   `param:"int16"`
		Int32   int32   `param:"int32"`
		Int64   int64   `param:"int64"`
		Uint    uint    `param:"uint"`
		Uint8   uint8   `param:"uint8"`
		Uint16  uint16  `param:"uint16"`
		Uint32  uint32  `param:"uint32"`
		Uint64  uint64  `param:"uint64"`
		Float32 float32 `param:"float32"`
		Float64 float64 `param:"float64"`
		String  string  `param:"string"`
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodGet,
		"/foobar"+
			"?bool=true"+
			"&int=1"+
			"&int8=1"+
			"&int16=1"+
			"&int32=1"+
			"&int64=1"+
			"&uint=1"+
			"&uint8=1"+
			"&uint16=1"+
			"&uint32=1"+
			"&uint64=1"+
			"&float32=1"+
			"&float64=1"+
			"&string=foobar",
		nil,
	)

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.True(t, f.Bool)
	assert.Equal(t, 1, f.Int)
	assert.Equal(t, int8(1), f.Int8)
	assert.Equal(t, int16(1), f.Int16)
	assert.Equal(t, int32(1), f.Int32)
	assert.Equal(t, int64(1), f.Int64)
	assert.Equal(t, uint(1), f.Uint)
	assert.Equal(t, uint8(1), f.Uint8)
	assert.Equal(t, uint16(1), f.Uint16)
	assert.Equal(t, uint32(1), f.Uint32)
	assert.Equal(t, uint64(1), f.Uint64)
	assert.Equal(t, float32(1), f.Float32)
	assert.Equal(t, float64(1), f.Float64)
	assert.Equal(t, "foobar", f.String)
}

func TestBindHEADNoBody(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Bool    bool
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Uint64  uint64
		Float32 float32
		Float64 float64
		String  string
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodHead,
		"/foobar"+
			"?Bool=true"+
			"&Int=1"+
			"&Int8=1"+
			"&Int16=1"+
			"&Int32=1"+
			"&Int64=1"+
			"&Uint=1"+
			"&Uint8=1"+
			"&Uint16=1"+
			"&Uint32=1"+
			"&Uint64=1"+
			"&Float32=1"+
			"&Float64=1"+
			"&String=foobar",
		nil,
	)

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.True(t, f.Bool)
	assert.Equal(t, 1, f.Int)
	assert.Equal(t, int8(1), f.Int8)
	assert.Equal(t, int16(1), f.Int16)
	assert.Equal(t, int32(1), f.Int32)
	assert.Equal(t, int64(1), f.Int64)
	assert.Equal(t, uint(1), f.Uint)
	assert.Equal(t, uint8(1), f.Uint8)
	assert.Equal(t, uint16(1), f.Uint16)
	assert.Equal(t, uint32(1), f.Uint32)
	assert.Equal(t, uint64(1), f.Uint64)
	assert.Equal(t, float32(1), f.Float32)
	assert.Equal(t, float64(1), f.Float64)
	assert.Equal(t, "foobar", f.String)
}

func TestBindDELETENoBody(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Bool    bool
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Uint64  uint64
		Float32 float32
		Float64 float64
		String  string
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodDelete,
		"/foobar"+
			"?bool=true"+
			"&int=1"+
			"&int8=1"+
			"&int16=1"+
			"&int32=1"+
			"&int64=1"+
			"&uint=1"+
			"&uint8=1"+
			"&uint16=1"+
			"&uint32=1"+
			"&uint64=1"+
			"&float32=1"+
			"&float64=1"+
			"&string=foobar",
		nil,
	)

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.True(t, f.Bool)
	assert.Equal(t, 1, f.Int)
	assert.Equal(t, int8(1), f.Int8)
	assert.Equal(t, int16(1), f.Int16)
	assert.Equal(t, int32(1), f.Int32)
	assert.Equal(t, int64(1), f.Int64)
	assert.Equal(t, uint(1), f.Uint)
	assert.Equal(t, uint8(1), f.Uint8)
	assert.Equal(t, uint16(1), f.Uint16)
	assert.Equal(t, uint32(1), f.Uint32)
	assert.Equal(t, uint64(1), f.Uint64)
	assert.Equal(t, float32(1), f.Float32)
	assert.Equal(t, float64(1), f.Float64)
	assert.Equal(t, "foobar", f.String)
}

func TestBindJSON(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		strings.NewReader(`{"foo": "bar", "bar": "foo"}`),
	)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

func TestBindXML(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `xml:"Foo"`
		Bar string `xml:"Bar"`
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		strings.NewReader(
			"<Foobar><Foo>bar</Foo><Bar>foo</Bar></Foobar>",
		),
	)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

func TestBindProtobuf(t *testing.T) {
	a := New()
	b := a.binder

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		bytes.NewReader([]byte{10, 6, 102, 111, 111, 98, 97, 114}),
	)
	req.Header.Set("Content-Type", "application/protobuf")

	sv := wrapperspb.StringValue{}
	assert.NoError(t, b.bind(&sv, req))
	assert.Equal(t, "foobar", sv.Value)
}

func TestBindMsgpack(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `msgpack:"foo"`
		Bar string `msgpack:"bar"`
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		bytes.NewReader([]byte{
			130, 163, 102, 111,
			111, 163, 98, 97,
			114, 163, 98, 97,
			114, 163, 102, 111,
			111,
		}),
	)
	req.Header.Set("Content-Type", "application/msgpack")

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

func TestBindTOML(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `toml:"foo"`
		Bar string `toml:"bar"`
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		strings.NewReader("foo=\"bar\"\nbar=\"foo\""),
	)
	req.Header.Set("Content-Type", "application/toml; charset=utf-8")

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

func TestBindYAML(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `yaml:"foo"`
		Bar string `yaml:"bar"`
	}

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		strings.NewReader("foo: \"bar\"\nbar: \"foo\""),
	)
	req.Header.Set("Content-Type", "application/yaml; charset=utf-8")

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

func TestBindXWWWFormURLEncoded(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `param:"foo"`
		Bar string `param:"bar"`
	}

	vs := url.Values{}
	vs.Set("foo", "bar")
	vs.Set("bar", "foo")

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		strings.NewReader(vs.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded; charset=utf-8",
	)

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

func TestBindFormData(t *testing.T) {
	a := New()
	b := a.binder

	type foobar struct {
		Foo string `param:"foo"`
		Bar string `param:"bar"`
	}

	buf := bytes.Buffer{}
	mpw := multipart.NewWriter(&buf)
	mpw.WriteField("foo", "bar")
	mpw.WriteField("bar", "foo")
	mpw.Close()

	req, _, _ := fakeRRCycle(
		a,
		http.MethodPost,
		"/foobar",
		bytes.NewReader(buf.Bytes()),
	)
	req.Header.Set("Content-Type", mpw.FormDataContentType())

	f := foobar{}
	assert.NoError(t, b.bind(&f, req))
	assert.Equal(t, "bar", f.Foo)
	assert.Equal(t, "foo", f.Bar)
}

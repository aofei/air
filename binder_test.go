package air

import (
	"encoding/xml"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinderBindError(t *testing.T) {
	a := New()
	b := a.Binder.(*binder)
	c := NewContext(a)

	req, _ := http.NewRequest("GET", "/", nil)

	c.feed(req, nil)

	assert.Error(t, b.Bind(&Map{}, c.Request))

	req, _ = http.NewRequest("POST", "/", nil)

	c.reset()
	c.feed(req, nil)

	assert.Error(t, b.Bind(&Map{}, c.Request))

	req, _ = http.NewRequest(
		"POST",
		"/",
		strings.NewReader("{\"num\":999e999}"),
	)
	req.Header.Set("Content-Type", "application/json")

	c.reset()
	c.feed(req, nil)

	assert.Error(t, b.Bind(&Map{}, c.Request))

	req, _ = http.NewRequest("POST", "/", strings.NewReader("{,}"))
	req.Header.Set("Content-Type", "application/json")

	c.reset()
	c.feed(req, nil)

	assert.Error(t, b.Bind(&Map{}, c.Request))

	x := xml.Header + `<Info>
		<Num>1</Num>`

	req, _ = http.NewRequest("POST", "/", strings.NewReader(x))
	req.Header.Set("Content-Type", "application/xml")

	c.reset()
	c.feed(req, nil)

	assert.Error(t, b.Bind(&struct {
		Num int
	}{}, c.Request))

	x = xml.Header + `<Info>
		<Num>foobar</Num>
	</Info>`

	req, _ = http.NewRequest("POST", "/", strings.NewReader(x))
	req.Header.Set("Content-Type", "application/xml")

	c.reset()
	c.feed(req, nil)

	assert.Error(t, b.Bind(&struct {
		Num int
	}{}, c.Request))
}

func TestBinderBindDataError(t *testing.T) {
	a := New()
	b := a.Binder.(*binder)
	assert.Error(t, b.bindData(&Map{}, nil, ""))
}

func TestBinderSetWithProperTypeError(t *testing.T) {
	var c complex64
	k := reflect.TypeOf(c).Kind()
	v := reflect.ValueOf(c)
	assert.Error(t, setWithProperType(k, "", v))
}

func TestBinderSetMethods(t *testing.T) {
	s := &struct {
		Int   int
		Uint  uint
		Bool  bool
		Float float64
	}{}

	v := reflect.ValueOf(s).Elem()

	assert.NoError(t, setIntField("", 64, v.Field(0)))
	assert.NoError(t, setUintField("", 64, v.Field(1)))
	assert.NoError(t, setBoolField("", v.Field(2)))
	assert.NoError(t, setFloatField("", 64, v.Field(3)))
}

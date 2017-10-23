package air

import (
	"encoding/xml"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinderBindError(t *testing.T) {
	a := New()
	b := a.binder

	req := newRequest(a, httptest.NewRequest("GET", "/", nil))
	assert.Error(t, b.bind(&map[string]interface{}{}, req))

	req = newRequest(a, httptest.NewRequest("POST", "/", nil))
	assert.Error(t, b.bind(&map[string]interface{}{}, req))

	req = newRequest(a, httptest.NewRequest(
		"POST",
		"/",
		strings.NewReader("{\"num\":999e999}"),
	))
	req.Headers["Content-Type"] = "application/json"
	assert.Error(t, b.bind(&map[string]interface{}{}, req))

	req = newRequest(a, httptest.NewRequest(
		"POST",
		"/",
		strings.NewReader("{,}"),
	))
	req.Headers["Content-Type"] = "application/json"
	assert.Error(t, b.bind(&map[string]interface{}{}, req))

	req = newRequest(a, httptest.NewRequest(
		"POST",
		"/",
		strings.NewReader(xml.Header+"<Info>\n<Num>1</Num>"),
	))
	req.Headers["Content-Type"] = "application/xml"
	assert.Error(t, b.bind(&struct{ Num int }{}, req))

	req = newRequest(a, httptest.NewRequest(
		"POST",
		"/",
		strings.NewReader(
			xml.Header+"<Info>\n<Num>foobar</Num>\n</Info>",
		),
	))
	req.Headers["Content-Type"] = "application/xml"
	assert.Error(t, b.bind(&struct{ Num int }{}, req))
}

func TestBinderBindValuesError(t *testing.T) {
	a := New()
	b := a.binder
	assert.Error(t, b.bindValues(&map[string]interface{}{}, nil, ""))
}

func TestBinderSetWithProperTypeError(t *testing.T) {
	var c complex64
	k := reflect.TypeOf(c).Kind()
	v := reflect.ValueOf(c)
	assert.Error(t, setWithProperType(k, "", v))
}

func TestBinderSetMethods(t *testing.T) {
	v := reflect.ValueOf(&struct {
		Int   int
		Uint  uint
		Bool  bool
		Float float64
	}{}).Elem()

	assert.NoError(t, setIntField("", 64, v.Field(0)))
	assert.NoError(t, setUintField("", 64, v.Field(1)))
	assert.NoError(t, setBoolField("", v.Field(2)))
	assert.NoError(t, setFloatField("", 64, v.Field(3)))
}

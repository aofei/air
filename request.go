package air

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Request is an HTTP request.
type Request struct {
	Air           *Air
	Method        string
	Scheme        string
	Authority     string
	Path          string
	Header        http.Header
	Body          io.Reader
	ContentLength int64
	Values        map[string]interface{}

	hr              *http.Request
	res             *Response
	params          []*RequestParam
	parseParamsOnce *sync.Once
	localizedString func(string) string
}

// HTTPRequest returns the underlying `http.Request` of the r.
//
// ATTENTION: You should never call this method unless you know what you are
// doing. And, be sure to call the `Request#SetHTTPRequest()` when you have
// modified it.
func (r *Request) HTTPRequest() *http.Request {
	r.hr.Method = r.Method
	r.hr.Host = r.Authority
	r.hr.RequestURI = r.Path
	r.hr.Body = r.Body.(io.ReadCloser)
	r.hr.ContentLength = r.ContentLength
	return r.hr
}

// SetHTTPRequest sets the r to the r's underlying `http.Request`.
//
// ATTENTION: You should never call this method unless you know what you are
// doing.
func (r *Request) SetHTTPRequest(hr *http.Request) {
	r.Method = hr.Method
	r.Scheme = "http"
	if hr.TLS != nil {
		r.Scheme = "https"
	}

	r.Authority = hr.Host
	r.Path = hr.RequestURI
	r.Body = hr.Body
	r.ContentLength = hr.ContentLength
	r.hr = hr
}

// RemoteAddress returns the last network address that sent the r.
func (r *Request) RemoteAddress() string {
	return r.hr.RemoteAddr
}

// ClientAddress returns the original network address that sent the r.
func (r *Request) ClientAddress() string {
	ca := r.RemoteAddress()
	if f := r.Header.Get("Forwarded"); f != "" { // See RFC 7239
		for _, p := range strings.Split(strings.Split(f, ",")[0], ";") {
			p := strings.TrimSpace(p)
			if strings.HasPrefix(p, "for=") {
				ca = strings.TrimSuffix(
					strings.TrimPrefix(p[4:], "\"["),
					"]\"",
				)
				break
			}
		}
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ca = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	return ca
}

// Cookie returns the matched `http.Cookie` for the name. It returns nil if not
// found.
func (r *Request) Cookie(name string) *http.Cookie {
	c, _ := r.hr.Cookie(name)
	return c
}

// Cookies returns all the `http.Cookie` in the r.
func (r *Request) Cookies() []*http.Cookie {
	return r.hr.Cookies()
}

// Param returns the matched `RequestParam` for the name. It returns nil if not
// found.
func (r *Request) Param(name string) *RequestParam {
	r.parseParamsOnce.Do(r.parseParams)

	for _, p := range r.params {
		if p.Name == name {
			return p
		}
	}

	return nil
}

// Params returns all the `RequestParam` in the r.
func (r *Request) Params() []*RequestParam {
	r.parseParamsOnce.Do(r.parseParams)
	return r.params
}

// parseParams parses the params sent with the r into the `r.params`.
func (r *Request) parseParams() {
	if r.hr.Form == nil || r.hr.MultipartForm == nil {
		r.hr.ParseMultipartForm(32 << 20)
	}

FormLoop:
	for n, vs := range r.hr.Form {
		pvs := make([]*RequestParamValue, 0, len(vs))
		for _, v := range vs {
			pvs = append(pvs, &RequestParamValue{
				i: v,
			})
		}

		for _, p := range r.params {
			if p.Name == n {
				p.Values = append(p.Values, pvs...)
				continue FormLoop
			}
		}

		r.params = append(r.params, &RequestParam{
			Name:   n,
			Values: pvs,
		})
	}

	if r.hr.MultipartForm != nil {
	MultipartFormValueLoop:
		for n, vs := range r.hr.MultipartForm.Value {
			pvs := make([]*RequestParamValue, 0, len(vs))
			for _, v := range vs {
				pvs = append(pvs, &RequestParamValue{
					i: v,
				})
			}

			for _, p := range r.params {
				if p.Name == n {
					p.Values = append(p.Values, pvs...)
					continue MultipartFormValueLoop
				}
			}

			r.params = append(r.params, &RequestParam{
				Name:   n,
				Values: pvs,
			})
		}

	MultipartFormFileLoop:
		for n, vs := range r.hr.MultipartForm.File {
			pvs := make([]*RequestParamValue, 0, len(vs))
			for _, v := range vs {
				pvs = append(pvs, &RequestParamValue{
					i: v,
				})
			}

			for _, p := range r.params {
				if p.Name == n {
					p.Values = append(p.Values, pvs...)
					continue MultipartFormFileLoop
				}
			}

			r.params = append(r.params, &RequestParam{
				Name:   n,
				Values: pvs,
			})
		}
	}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return r.Air.binder.bind(v, r)
}

// LocalizedString returns localized string for the key.
//
// It only works if the `I18nEnabled` is true.
func (r *Request) LocalizedString(key string) string {
	if r.localizedString == nil {
		r.Air.i18n.localize(r)
	}

	return r.localizedString(key)
}

// RequestParam is an HTTP request param.
type RequestParam struct {
	Name   string
	Values []*RequestParamValue
}

// Value returns the first value of the rp. It returns nil if the rp is nil or
// there are no values.
func (rp *RequestParam) Value() *RequestParamValue {
	if rp == nil || len(rp.Values) == 0 {
		return nil
	}

	return rp.Values[0]
}

// RequestParamValue is an HTTP request param value.
type RequestParamValue struct {
	i    interface{}
	b    *bool
	i64  *int64
	ui64 *uint64
	f64  *float64
	s    *string
	f    *multipart.FileHeader
}

// Bool returns a `bool` from the rpv's underlying value.
func (rpv *RequestParamValue) Bool() (bool, error) {
	if rpv.b == nil {
		b, err := strconv.ParseBool(rpv.String())
		if err != nil {
			return false, err
		}

		rpv.b = &b
	}

	return *rpv.b, nil
}

// Int returns an `int` from the rpv's underlying value.
func (rpv *RequestParamValue) Int() (int, error) {
	if rpv.i64 == nil {
		i64, err := strconv.ParseInt(rpv.String(), 10, 0)
		if err != nil {
			return 0, err
		}

		rpv.i64 = &i64
	}

	return int(*rpv.i64), nil
}

// Int8 returns an `int8` from the rpv's underlying value.
func (rpv *RequestParamValue) Int8() (int8, error) {
	if rpv.i64 == nil {
		i64, err := strconv.ParseInt(rpv.String(), 10, 8)
		if err != nil {
			return 0, err
		}

		rpv.i64 = &i64
	}

	return int8(*rpv.i64), nil
}

// Int16 returns an `int16` from the rpv's underlying value.
func (rpv *RequestParamValue) Int16() (int16, error) {
	if rpv.i64 == nil {
		i64, err := strconv.ParseInt(rpv.String(), 10, 16)
		if err != nil {
			return 0, err
		}

		rpv.i64 = &i64
	}

	return int16(*rpv.i64), nil
}

// Int32 returns an `int32` from the rpv's underlying value.
func (rpv *RequestParamValue) Int32() (int32, error) {
	if rpv.i64 == nil {
		i64, err := strconv.ParseInt(rpv.String(), 10, 32)
		if err != nil {
			return 0, err
		}

		rpv.i64 = &i64
	}

	return int32(*rpv.i64), nil
}

// Int64 returns an `int64` from the rpv's underlying value.
func (rpv *RequestParamValue) Int64() (int64, error) {
	if rpv.i64 == nil {
		i64, err := strconv.ParseInt(rpv.String(), 10, 64)
		if err != nil {
			return 0, err
		}

		rpv.i64 = &i64
	}

	return *rpv.i64, nil
}

// Uint returns an `uint` from the rpv's underlying value.
func (rpv *RequestParamValue) Uint() (uint, error) {
	if rpv.ui64 == nil {
		ui64, err := strconv.ParseUint(rpv.String(), 10, 0)
		if err != nil {
			return 0, err
		}

		rpv.ui64 = &ui64
	}

	return uint(*rpv.ui64), nil
}

// Uint8 returns an `uint8` from the rpv's underlying value.
func (rpv *RequestParamValue) Uint8() (uint8, error) {
	if rpv.ui64 == nil {
		ui64, err := strconv.ParseUint(rpv.String(), 10, 8)
		if err != nil {
			return 0, err
		}

		rpv.ui64 = &ui64
	}

	return uint8(*rpv.ui64), nil
}

// Uint16 returns an `uint16` from the rpv's underlying value.
func (rpv *RequestParamValue) Uint16() (uint16, error) {
	if rpv.ui64 == nil {
		ui64, err := strconv.ParseUint(rpv.String(), 10, 16)
		if err != nil {
			return 0, err
		}

		rpv.ui64 = &ui64
	}

	return uint16(*rpv.ui64), nil
}

// Uint32 returns an `uint32` from the rpv's underlying value.
func (rpv *RequestParamValue) Uint32() (uint32, error) {
	if rpv.ui64 == nil {
		ui64, err := strconv.ParseUint(rpv.String(), 10, 32)
		if err != nil {
			return 0, err
		}

		rpv.ui64 = &ui64
	}

	return uint32(*rpv.ui64), nil
}

// Uint64 returns an `uint64` from the rpv's underlying value.
func (rpv *RequestParamValue) Uint64() (uint64, error) {
	if rpv.ui64 == nil {
		ui64, err := strconv.ParseUint(rpv.String(), 10, 64)
		if err != nil {
			return 0, err
		}

		rpv.ui64 = &ui64
	}

	return *rpv.ui64, nil
}

// Float32 returns a `float32` from the rpv's underlying value.
func (rpv *RequestParamValue) Float32() (float32, error) {
	if rpv.f64 == nil {
		f64, err := strconv.ParseFloat(rpv.String(), 32)
		if err != nil {
			return 0, err
		}

		rpv.f64 = &f64
	}

	return float32(*rpv.f64), nil
}

// Float64 returns a `float64` from the rpv's underlying value.
func (rpv *RequestParamValue) Float64() (float64, error) {
	if rpv.f64 == nil {
		f64, err := strconv.ParseFloat(rpv.String(), 64)
		if err != nil {
			return 0, err
		}

		rpv.f64 = &f64
	}

	return *rpv.f64, nil
}

// String returns a `string` from the rpv's underlying value.
func (rpv *RequestParamValue) String() string {
	if rpv.s == nil {
		if s, ok := rpv.i.(string); ok {
			rpv.s = &s
		} else {
			s := fmt.Sprintf("%v", rpv.i)
			rpv.s = &s
		}
	}

	return *rpv.s
}

// File returns a `multipart.FileHeader` from the rpv's underlying value.
func (rpv *RequestParamValue) File() (*multipart.FileHeader, error) {
	if rpv.f == nil {
		fh, ok := rpv.i.(*multipart.FileHeader)
		if !ok {
			return nil, http.ErrMissingFile
		}

		rpv.f = fh
	}

	return rpv.f, nil
}

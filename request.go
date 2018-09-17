package air

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Request is an HTTP request.
type Request struct {
	Method        string
	URL           *URL
	Proto         string
	Headers       map[string]string
	Body          io.Reader
	ContentLength int64
	Cookies       map[string]*Cookie
	Params        map[string]*RequestParamValue
	RemoteAddr    string
	ClientIP      net.IP
	Values        map[string]interface{}

	httpRequest     *http.Request
	parsedCookies   bool
	parsedParams    bool
	parsedFiles     bool
	localizedString func(string) string
}

// ParseCookies parses the cookies sent with the r into the `r.Cookies`.
//
// It will be called after routing. Relax, you can of course call it before
// routing, it will only take effect on the very first call.
func (r *Request) ParseCookies() {
	if r.parsedCookies {
		return
	}

	r.parsedCookies = true

	for _, line := range r.httpRequest.Header["Cookie"] {
		ps := strings.Split(strings.TrimSpace(line), ";")
		if len(ps) == 1 && ps[0] == "" {
			continue
		}

		for i := 0; i < len(ps); i++ {
			ps[i] = strings.TrimSpace(ps[i])
			if len(ps[i]) == 0 {
				continue
			}

			n, v := ps[i], ""
			if i := strings.Index(n, "="); i >= 0 {
				n, v = n[:i], n[i+1:]
			}

			if !validCookieName(n) {
				continue
			}

			if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}

			if !validCookieValue(v) {
				continue
			}

			r.Cookies[n] = &Cookie{
				Name:  n,
				Value: v,
			}
		}
	}
}

// ParseParams parses the params sent with the r into the `r.Params`.
//
// It will be called after routing. Relax, you can of course call it before
// routing, it will only take effect on the very first call.
func (r *Request) ParseParams() {
	if r.parsedParams {
		return
	}

	r.parsedParams = true

	if r.httpRequest.Form == nil || r.httpRequest.MultipartForm == nil {
		r.httpRequest.ParseMultipartForm(32 << 20)
	}

	for k, v := range r.httpRequest.Form {
		if len(v) > 0 {
			r.Params[k] = &RequestParamValue{
				i: v[0],
			}
		}
	}

	if r.httpRequest.MultipartForm != nil {
		for k, v := range r.httpRequest.MultipartForm.Value {
			if len(v) > 0 {
				r.Params[k] = &RequestParamValue{
					i: v[0],
				}
			}
		}

		for k, v := range r.httpRequest.MultipartForm.File {
			if len(v) > 0 {
				r.Params[k] = &RequestParamValue{
					i: v[0],
				}
			}
		}
	}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return theBinder.bind(v, r)
}

// LocalizedString returns localized string for the provided key.
//
// It only works if the `I18nEnabled` is true.
func (r *Request) LocalizedString(key string) string {
	if r.localizedString != nil {
		return r.localizedString(key)
	}

	return key
}

// RequestParamValue is an HTTP request param value.
type RequestParamValue struct {
	i    interface{}
	b    *bool
	i64  *int64
	ui64 *uint64
	f64  *float64
	s    *string
	f    *RequestParamFileValue
}

// Interface returns the rpv's underlying value.
func (rpv *RequestParamValue) Interface() interface{} {
	return rpv.i
}

// Bool tries to returns a `bool` from the rpv's underlying value.
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

// Int tries to returns an `int` from the rpv's underlying value.
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

// Int8 tries to returns an `int8` from the rpv's underlying value.
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

// Int16 tries to returns an `int16` from the rpv's underlying value.
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

// Int32 tries to returns an `int32` from the rpv's underlying value.
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

// Int64 tries to returns an `int64` from the rpv's underlying value.
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

// Uint tries to returns an `uint` from the rpv's underlying value.
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

// Uint8 tries to returns an `uint8` from the rpv's underlying value.
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

// Uint16 tries to returns an `uint16` from the rpv's underlying value.
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

// Uint32 tries to returns an `uint32` from the rpv's underlying value.
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

// Uint64 tries to returns an `uint64` from the rpv's underlying value.
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

// Float32 tries to returns a `float32` from the rpv's underlying value.
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

// Float64 tries to returns a `float64` from the rpv's underlying value.
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

// String tries to returns a `string` from the rpv's underlying value.
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

// File tries to returns a `RequestParamFileValue` from the rpv's underlying
// value.
func (rpv *RequestParamValue) File() (*RequestParamFileValue, error) {
	if rpv.f == nil {
		fh, ok := rpv.i.(*multipart.FileHeader)
		if !ok {
			return nil, errors.New("not a param file value")
		}

		f, err := fh.Open()
		if err != nil {
			return nil, err
		}

		rpv.f = &RequestParamFileValue{
			Reader:   f,
			Seeker:   f,
			Closer:   f,
			Filename: fh.Filename,
			Headers:  make(map[string]string, len(fh.Header)),
		}

		for k, v := range fh.Header {
			if len(v) > 0 {
				rpv.f.Headers[k] = strings.Join(v, ", ")
			}
		}

		rpv.f.Size, _ = f.Seek(0, io.SeekEnd)
		f.Seek(0, io.SeekStart)
	}

	return rpv.f, nil
}

// RequestParamFileValue is an HTTP request param file value.
type RequestParamFileValue struct {
	io.Reader
	io.Seeker
	io.Closer

	Filename string
	Headers  map[string]string
	Size     int64
}

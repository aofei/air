package air

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Request is an HTTP request.
type Request struct {
	Method        string
	Scheme        string
	Authority     string
	Path          string
	Headers       map[string]*Header
	Body          io.Reader
	ContentLength int64
	Cookies       map[string]*Cookie
	Params        map[string]*RequestParam
	RemoteAddress string
	ClientAddress string
	Values        map[string]interface{}

	request          *http.Request
	localizedString  func(string) string
	parseCookiesOnce *sync.Once
	parseParamsOnce  *sync.Once
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return theBinder.bind(v, r)
}

// LocalizedString returns localized string for the key.
//
// It only works if the `I18nEnabled` is true.
func (r *Request) LocalizedString(key string) string {
	if r.localizedString != nil {
		return r.localizedString(key)
	}

	return key
}

// ParseCookies parses the cookies sent with the r into the `r.Cookies`.
//
// It will be called after routing. Relax, you can of course call it before
// routing, it will only take effect on the very first call.
func (r *Request) ParseCookies() {
	r.parseCookiesOnce.Do(func() {
		ch := r.Headers["cookie"]
		if ch == nil {
			return
		}

		for _, c := range ch.Values {
			ps := strings.Split(strings.TrimSpace(c), ";")
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

				if len(v) > 1 && v[0] == '"' &&
					v[len(v)-1] == '"' {
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
	})
}

// ParseParams parses the params sent with the r into the `r.Params`.
//
// It will be called after routing. Relax, you can of course call it before
// routing, it will only take effect on the very first call.
func (r *Request) ParseParams() {
	r.parseParamsOnce.Do(func() {
		if r.request.Form == nil || r.request.MultipartForm == nil {
			r.request.ParseMultipartForm(32 << 20)
		}

		for n, vs := range r.request.Form {
			pvs := make([]*RequestParamValue, 0, len(vs))
			for _, v := range vs {
				pvs = append(pvs, &RequestParamValue{
					i: v,
				})
			}

			if r.Params[n] == nil {
				r.Params[n] = &RequestParam{
					Name:   n,
					Values: pvs,
				}
			} else {
				r.Params[n].Values = append(
					r.Params[n].Values,
					pvs...,
				)
			}
		}

		if r.request.MultipartForm != nil {
			for n, vs := range r.request.MultipartForm.Value {
				pvs := make([]*RequestParamValue, 0, len(vs))
				for _, v := range vs {
					pvs = append(pvs, &RequestParamValue{
						i: v,
					})
				}

				if r.Params[n] == nil {
					r.Params[n] = &RequestParam{
						Name:   n,
						Values: pvs,
					}
				} else {
					r.Params[n].Values = append(
						r.Params[n].Values,
						pvs...,
					)
				}
			}

			for n, vs := range r.request.MultipartForm.File {
				pvs := make([]*RequestParamValue, 0, len(vs))
				for _, v := range vs {
					pvs = append(pvs, &RequestParamValue{
						i: v,
					})
				}

				if r.Params[n] == nil {
					r.Params[n] = &RequestParam{
						Name:   n,
						Values: pvs,
					}
				} else {
					r.Params[n].Values = append(
						r.Params[n].Values,
						pvs...,
					)
				}
			}
		}
	})
}

// RequestParam is an HTTP request param.
type RequestParam struct {
	Name   string
	Values []*RequestParamValue
}

// FirstValue returns the first value of the rp. It returns nil if the rp is nil
// or there are no values.
func (rp *RequestParam) FirstValue() *RequestParamValue {
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
	f    *RequestParamFileValue
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

// File returns a `RequestParamFileValue` from the rpv's underlying value.
func (rpv *RequestParamValue) File() (*RequestParamFileValue, error) {
	if rpv.f == nil {
		fh, ok := rpv.i.(*multipart.FileHeader)
		if !ok {
			return nil, errors.New("not a request param file value")
		}

		rpv.f = &RequestParamFileValue{
			Filename: fh.Filename,
			Headers:  make(map[string]*Header, len(fh.Header)),

			fh: fh,
		}

		for n, vs := range fh.Header {
			h := &Header{
				Name:   strings.ToLower(n),
				Values: vs,
			}

			rpv.f.Headers[h.Name] = h
		}

		if s := reflect.ValueOf(fh).FieldByName("Size"); !s.IsNil() {
			rpv.f.ContentLength = s.Int()
		} else {
			rpv.f.ContentLength, _ = rpv.f.Seek(0, io.SeekEnd)
			rpv.f.Seek(0, io.SeekStart)
		}
	}

	return rpv.f, nil
}

// RequestParamFileValue is an HTTP request param file value.
type RequestParamFileValue struct {
	Filename      string
	Headers       map[string]*Header
	ContentLength int64

	fh *multipart.FileHeader
	f  multipart.File
}

// Read implements the `io.Reader`.
func (v *RequestParamFileValue) Read(b []byte) (int, error) {
	if v.f == nil {
		var err error
		if v.f, err = v.fh.Open(); err != nil {
			return 0, err
		}
	}

	return v.f.Read(b)
}

// Seek implements the `io.Seeker`.
func (v *RequestParamFileValue) Seek(offset int64, whence int) (int64, error) {
	if v.f == nil {
		var err error
		if v.f, err = v.fh.Open(); err != nil {
			return 0, err
		}
	}

	return v.f.Seek(offset, whence)
}

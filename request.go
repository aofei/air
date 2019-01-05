package air

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Request is an HTTP request.
type Request struct {
	// Air is where the current request belong.
	Air *Air

	// Method is the method of the current request.
	//
	// See RFC 7231, section 4.3.
	//
	// For HTTP/1.x, it is from the request-line.
	//
	// For HTTP/2, it is from the ":method" pseudo-header.
	Method string

	// Scheme is the scheme of the current request, it is "http" or "https".
	//
	// See RFC 3986, section 3.1.
	//
	// For HTTP/1.x, it is from the request-line.
	//
	// For HTTP/2, it is from the ":scheme" pseudo-header.
	Scheme string

	// Authority is the authority of the current request. It may be of the
	// form "host:port".
	//
	// See RFC 3986, Section 3.2.
	//
	// For HTTP/1.x, it is from the "Host" header.
	//
	// For HTTP/2, it is from the ":authority" pseudo-header.
	Authority string

	// Path is the path of the current request.
	//
	// For HTTP/1.x, it represents the request-target of the request-line.
	// See RFC 7230, section 3.1.1.
	//
	// For HTTP/2, it represents the ":path" pseudo-header. See RFC 7540,
	// section 8.1.2.3.
	Path string

	// Header is the header key-value pair map of the current request.
	//
	// See RFC 7231, section 5.
	//
	// It is basically the same for both of HTTP/1.x and HTTP/2. The only
	// difference is that HTTP/2 requires header names to be lowercase. See
	// RFC 7540, section 8.1.2.
	Header http.Header

	// Body is the message body of the current request.
	Body io.Reader

	// ContentLength records the length of the associated content. The value
	// -1 indicates that the length is unknown. Values >= 0 indicate that
	// the given number of bytes may be read from the `Body`.
	ContentLength int64

	// Context is the context that associated with the current request.
	//
	// It is canceled when the client's connection closes, the current
	// request is canceled (with HTTP/2), or when the current
	// request-response cycle is finished.
	Context context.Context

	hr                   *http.Request
	res                  *Response
	params               []*RequestParam
	routeParamNames      []string
	routeParamValues     []string
	parseRouteParamsOnce *sync.Once
	parseOtherParamsOnce *sync.Once
	localizedString      func(string) string
}

// HTTPRequest returns the underlying `http.Request` of the r.
//
// ATTENTION: You should never call this method unless you know what you are
// doing. And, be sure to call the `r#SetHTTPRequest()` when you have modified
// it.
func (r *Request) HTTPRequest() *http.Request {
	r.hr.Method = r.Method
	r.hr.Host = r.Authority
	if r.hr.RequestURI != r.Path {
		r.hr.URL, _ = url.ParseRequestURI(r.Path)
	}

	r.hr.RequestURI = r.Path
	r.hr.Header = r.Header
	r.hr.Body = r.Body.(io.ReadCloser)
	r.hr.ContentLength = r.ContentLength
	if r.hr.Context() != r.Context {
		r.hr = r.hr.WithContext(r.Context)
	}

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
	r.Header = hr.Header
	r.Body = hr.Body
	r.ContentLength = hr.ContentLength
	r.Context = hr.Context()
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
	if r.routeParamNames != nil {
		r.parseRouteParamsOnce.Do(r.parseRouteParams)
	}

	r.parseOtherParamsOnce.Do(r.parseOtherParams)

	for _, p := range r.params {
		if p.Name == name {
			return p
		}
	}

	return nil
}

// Params returns all the `RequestParam` in the r.
func (r *Request) Params() []*RequestParam {
	if r.routeParamNames != nil {
		r.parseRouteParamsOnce.Do(r.parseRouteParams)
	}

	r.parseOtherParamsOnce.Do(r.parseOtherParams)

	return r.params
}

// parseRouteParams parses the route params sent with the r into the `r.params`.
func (r *Request) parseRouteParams() {
	r.growParams(len(r.routeParamNames))

RouteParamLoop:
	for i, pn := range r.routeParamNames {
		pv, err := url.PathUnescape(r.routeParamValues[i])
		if err != nil {
			continue
		}

		for _, p := range r.params {
			if p.Name != pn {
				continue
			}

			pvs := make([]*RequestParamValue, len(p.Values)+1)
			pvs[0] = &RequestParamValue{
				i: pv,
			}

			copy(pvs[1:], p.Values)
			p.Values = pvs

			continue RouteParamLoop
		}

		r.params = append(r.params, &RequestParam{
			Name: pn,
			Values: []*RequestParamValue{
				{
					i: pv,
				},
			},
		})
	}

	r.Air.router.routeParamValuesPool.Put(r.routeParamValues)
	r.routeParamNames = nil
	r.routeParamValues = nil
}

// parseOtherParams parses the other params sent with the r into the `r.params`.
func (r *Request) parseOtherParams() {
	if r.hr.Form == nil || r.hr.MultipartForm == nil {
		r.hr.ParseMultipartForm(32 << 20)
	}

	r.growParams(len(r.hr.Form))

FormLoop:
	for n, vs := range r.hr.Form {
		if len(vs) == 0 {
			continue
		}

		pvs := make([]*RequestParamValue, len(vs))
		for i, v := range vs {
			pvs[i] = &RequestParamValue{
				i: v,
			}
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

	if mf := r.hr.MultipartForm; mf != nil {
		r.growParams(len(mf.Value) + len(mf.File))

	MultipartFormValueLoop:
		for n, vs := range mf.Value {
			if len(vs) == 0 {
				continue
			}

			pvs := make([]*RequestParamValue, len(vs))
			for i, v := range vs {
				pvs[i] = &RequestParamValue{
					i: v,
				}
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
		for n, vs := range mf.File {
			if len(vs) == 0 {
				continue
			}

			pvs := make([]*RequestParamValue, len(vs))
			for i, v := range vs {
				pvs[i] = &RequestParamValue{
					i: v,
				}
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

// growParams grows `r.params`'s capacity, if necessary, to guarantee space for
// another n.
func (r *Request) growParams(n int) {
	if cap(r.params)-len(r.params) < n {
		ps := make([]*RequestParam, len(r.params), cap(r.params)+n)
		copy(ps, r.params)
		r.params = ps
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
	if !r.Air.I18nEnabled {
		return key
	}

	if r.localizedString == nil {
		r.Air.i18n.localize(r)
	}

	return r.localizedString(key)
}

// RequestParam is an HTTP request param.
type RequestParam struct {
	// Name is the name of the current request param.
	Name string

	// Values is the values of the current request param.
	//
	// The route param value always has the highest weight.
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
//
// It may represent a route param value, a request query value, a request form
// value, a request multipart form value or a request multipart form file value.
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
			s := fmt.Sprint(rpv.i)
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

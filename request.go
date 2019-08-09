package air

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Request is an HTTP request.
//
// The `Request` not only represents HTTP/1.x requests, but also represents
// HTTP/2 requests, and always acts as HTTP/2 requests.
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
	//
	// E.g.: "GET"
	Method string

	// Scheme is the scheme of the current request, it is "http" or "https".
	//
	// See RFC 3986, section 3.1.
	//
	// For HTTP/1.x, it is from the request-line.
	//
	// For HTTP/2, it is from the ":scheme" pseudo-header.
	//
	// E.g.: "http"
	Scheme string

	// Authority is the authority of the current request. It may be of the
	// form "host:port".
	//
	// See RFC 3986, Section 3.2.
	//
	// For HTTP/1.x, it is from the Host header.
	//
	// For HTTP/2, it is from the ":authority" pseudo-header.
	//
	// E.g.: "localhost:8080"
	Authority string

	// Path is the path of the current request. Note that it contains the
	// query part (anyway, the HTTP/2 specification says so).
	//
	// For HTTP/1.x, it represents the request-target of the request-line.
	// See RFC 7230, section 3.1.1.
	//
	// For HTTP/2, it represents the ":path" pseudo-header. See RFC 7540,
	// section 8.1.2.3.
	//
	// E.g.: "/foo/bar?foo=bar"
	Path string

	// Header is the header map of the current request.
	//
	// The values of the Trailer header are the names of the trailers which
	// will come later. In this case, those names of the header map will be
	// set after reading from the `Body` returns the `io.EOF`.
	//
	// See RFC 7231, section 5.
	//
	// The `Header` is basically the same for both HTTP/1.x and HTTP/2. The
	// only difference is that HTTP/2 requires header names to be lowercase
	// (for aesthetic reasons, this framework decided to follow this rule
	// implicitly, so please use the header name in the HTTP/1.x way). See
	// RFC 7540, section 8.1.2.
	//
	// E.g.: {"Foo": ["bar"]}
	Header http.Header

	// Body is the message body of the current request.
	Body io.Reader

	// ContentLength records the length of the associated content. The value
	// -1 indicates that the length is unknown (it will be set after reading
	// from the `Body` returns the `io.EOF`). Values >= 0 indicate that the
	// given number of bytes may be read from the `Body`.
	ContentLength int64

	// Context is the context that associated with the current request.
	//
	// The `Context` is canceled when the client's connection closes, the
	// current request is canceled (with HTTP/2), or when the current
	// request-response cycle is finished.
	Context context.Context

	hr                   *http.Request
	res                  *Response
	params               []*RequestParam
	routeParamNames      []string
	routeParamValues     []string
	parseRouteParamsOnce *sync.Once
	parseOtherParamsOnce *sync.Once
	values               map[string]interface{}
	localizedString      func(string) string
}

// HTTPRequest returns the underlying `http.Request` of the r.
//
// ATTENTION: You should never call this method unless you know what you are
// doing. And, be sure to call the `SetHTTPRequest` of the r when you have
// modified it.
func (r *Request) HTTPRequest() *http.Request {
	r.hr.Method = r.Method
	r.hr.Host = r.Authority
	if r.hr.RequestURI != r.Path {
		p, q := splitPathQuery(r.Path)
		if p != r.hr.URL.Path {
			r.hr.URL.Path = p
			r.hr.URL.RawPath = ""
		}

		r.hr.URL.ForceQuery = strings.HasSuffix(r.Path, "?") &&
			strings.Count(r.Path, "?") == 1
		r.hr.URL.RawQuery = q
	}

	r.hr.RequestURI = r.Path
	r.hr.Header = r.Header
	if b, ok := r.Body.(io.ReadCloser); ok {
		r.hr.Body = b
	} else {
		r.hr.Body = ioutil.NopCloser(r.Body)
	}

	r.hr.ContentLength = r.ContentLength
	if r.hr.Context() != r.Context {
		r.hr = r.hr.WithContext(r.Context)
	}

	return r.hr
}

// SetHTTPRequest sets the hr to the underlying `http.Request` of the r.
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
	if len(hr.Trailer) > 0 && r.Header.Get("Trailer") == "" {
		tns := make([]string, 0, len(hr.Trailer))
		for n := range hr.Trailer {
			tns = append(tns, n)
		}

		r.Header.Set("Trailer", strings.Join(tns, ", "))
	}

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
//
// Usually, the original network address is the same as the last network address
// that sent the r. But, the Forwarded header and the X-Forwarded-For header
// will be considered, which may affect the return value.
func (r *Request) ClientAddress() string {
	ca := r.RemoteAddress()
	if f := r.Header.Get("Forwarded"); f != "" { // See RFC 7239
		for _, p := range strings.Split(strings.Split(f, ",")[0], ";") {
			p := strings.TrimSpace(p)
			if strings.HasPrefix(strings.ToLower(p), "for=") {
				ca = p[4:]
				ca = strings.TrimPrefix(ca, `"`)
				ca = strings.TrimSuffix(ca, `"`)
				break
			}
		}
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ca = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	return ca
}

// Cookies returns all `http.Cookie` in the r.
func (r *Request) Cookies() []*http.Cookie {
	return r.hr.Cookies()
}

// Cookie returns the matched `http.Cookie` for the name. It returns nil if not
// found.
func (r *Request) Cookie(name string) *http.Cookie {
	c, _ := r.hr.Cookie(name)
	return c
}

// Params returns all `RequestParam` in the r.
func (r *Request) Params() []*RequestParam {
	if r.routeParamNames != nil {
		r.parseRouteParamsOnce.Do(r.parseRouteParams)
	}

	r.parseOtherParamsOnce.Do(r.parseOtherParams)

	return r.params
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

// parseRouteParams parses the route params sent with the r into the `r.params`.
func (r *Request) parseRouteParams() {
	r.growParams(len(r.routeParamNames))

RouteParamLoop:
	for i, pn := range r.routeParamNames {
		pv, _ := url.PathUnescape(r.routeParamValues[i])
		if pv == "" {
			pv = r.routeParamValues[i]
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
	if r.hr.Form == nil {
		r.hr.ParseForm()
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

	if r.hr.MultipartForm == nil {
		r.hr.ParseMultipartForm(32 << 20)
	}

	if r.hr.MultipartForm == nil {
		return
	}

	r.growParams(len(r.hr.MultipartForm.Value))

MultipartFormValueLoop:
	for n, vs := range r.hr.MultipartForm.Value {
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

	r.growParams(len(r.hr.MultipartForm.File))

MultipartFormFileLoop:
	for n, vs := range r.hr.MultipartForm.File {
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

// growParams grows the capacity of the `r.params`, if necessary, to guarantee
// space for another n.
func (r *Request) growParams(n int) {
	if cap(r.params)-len(r.params) >= n {
		return
	}

	ps := make([]*RequestParam, len(r.params), cap(r.params)+n)
	copy(ps, r.params)
	r.params = ps
}

// Values returns the values associated with the r.
//
// Note that the returned map is always non-nil.
func (r *Request) Values() map[string]interface{} {
	if r.values == nil {
		r.values = map[string]interface{}{}
	}

	return r.values
}

// Value returns the matched `interface{}` for the key from the values
// associated with the r. It returns nil if not found.
func (r *Request) Value(key string) interface{} {
	return r.Values()[key]
}

// SetValue sets the matched `interface{}` for the key from the values
// associated with the r to the value.
func (r *Request) SetValue(key string, value interface{}) {
	r.Values()[key] = value
}

// Bind binds the r into the v based on the Content-Type header.
//
// Supported MIME types:
//   * application/json
//   * application/xml
//   * application/protobuf
//   * application/msgpack
//   * application/toml
//   * application/yaml
//   * application/x-www-form-urlencoded
//   * multipart/form-data
func (r *Request) Bind(v interface{}) error {
	return r.Air.binder.bind(v, r)
}

// LocalizedString returns localized string for the key based on the
// Accept-Language header. It returns the key without any changes if the
// `I18nEnabled` of the `Air` of the r is false or something goes wrong.
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
//
// The param may come from the route params, the request query, the request
// form, the request multipart form.
type RequestParam struct {
	// Name is the name of the current request param.
	Name string

	// Values is the values of the current request param.
	//
	// Access order: route param value (always at the first) > request query
	// value(s) > request form value(s) > request multipart form value(s) >
	// request multipart form file(s).
	Values []*RequestParamValue
}

// Value returns the first value of the rp. It returns nil if the rp is nil or
// there are no values.
//
// Note that route params always have values.
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

// Bool returns a `bool` from the underlying value of the rpv.
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

// Int returns an `int` from the underlying value of the rpv.
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

// Int8 returns an `int8` from the underlying value of the rpv.
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

// Int16 returns an `int16` from the underlying value of the rpv.
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

// Int32 returns an `int32` from the underlying value of the rpv.
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

// Int64 returns an `int64` from the underlying value of the rpv.
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

// Uint returns an `uint` from the underlying value of the rpv.
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

// Uint8 returns an `uint8` from the underlying value of the rpv.
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

// Uint16 returns an `uint16` from the underlying value of the rpv.
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

// Uint32 returns an `uint32` from the underlying value of the rpv.
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

// Uint64 returns an `uint64` from the underlying value of the rpv.
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

// Float32 returns a `float32` from the underlying value of the rpv.
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

// Float64 returns a `float64` from the underlying value of the rpv.
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

// String returns a `string` from the underlying value of the rpv. It returns ""
// if the rpv is not text-based.
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

// File returns a `multipart.FileHeader` from the underlying value of the rpv.
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

// requestBody is used to tie the `Request.Body` and the `http.Request.Body`
// together.
type requestBody struct {
	sync.Mutex

	r      *Request
	hr     *http.Request
	rc     io.ReadCloser
	cl     int64
	sawEOF bool
}

// Read implements the `io.Reader`.
func (rb *requestBody) Read(b []byte) (n int, err error) {
	rb.Lock()
	defer rb.Unlock()

	if rb.sawEOF {
		err = io.EOF
		return
	}

	if rb.r.ContentLength < 0 {
		n, err = rb.rc.Read(b)
	} else if rl := rb.r.ContentLength - rb.cl; rl > 0 {
		if int64(len(b)) > rl {
			b = b[:rl]
		}

		n, err = rb.rc.Read(b)
	}

	rb.cl += int64(n)
	if err == nil && rb.r.ContentLength >= 0 &&
		rb.r.ContentLength-rb.cl <= 0 {
		if err = rb.rc.Close(); err != nil {
			return
		}

		err = io.EOF
	}

	if err == io.EOF {
		rb.sawEOF = true

		tns := strings.Split(rb.r.Header.Get("Trailer"), ", ")
		for _, tn := range tns {
			rb.r.Header[tn] = rb.hr.Trailer[tn]
		}

		if rb.r.ContentLength < 0 {
			rb.r.ContentLength = rb.cl
		}
	}

	return
}

// Close implements the `io.Closer`.
func (rb *requestBody) Close() error {
	return nil
}

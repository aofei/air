package air

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Request is an HTTP request.
type Request struct {
	Method        string
	Scheme        string
	Authority     string
	Path          string
	Headers       Headers
	Body          io.Reader
	ContentLength int64
	Cookies       map[string]*Cookie
	Params        RequestParams
	Files         RequestFiles
	RemoteAddr    string
	ClientIP      net.IP
	Values        map[string]interface{}

	httpRequest         *http.Request
	parsedCookies       bool
	parsedParamAndFiles bool
	localizedString     func(string) string
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

// ParseParamAndFiles parses the params and the files sent with the r into the
// `r.Params` and `r.Files`.
//
// It will be called after routing. Relax, you can of course call it before
// routing, it will only take effect on the very first call.
func (r *Request) ParseParamAndFiles() {
	if r.parsedParamAndFiles {
		return
	}

	r.parsedParamAndFiles = true

	if r.httpRequest.Form == nil || r.httpRequest.MultipartForm == nil {
		r.httpRequest.ParseMultipartForm(32 << 20)
	}

	for k, v := range r.httpRequest.Form {
		r.Params[k] = v
	}

	if r.httpRequest.MultipartForm != nil {
		for k, v := range r.httpRequest.MultipartForm.Value {
			r.Params[k] = v
		}

		for k, v := range r.httpRequest.MultipartForm.File {
			fs := make([]*RequestFile, 0, len(v))
			for _, v := range v {
				f, err := v.Open()
				if err != nil {
					continue
				}

				hs := make(Headers, len(v.Header))
				for k, v := range v.Header {
					hs.Set(k, v)
				}

				cl, _ := f.Seek(0, io.SeekEnd)
				f.Seek(0, io.SeekStart)

				fs = append(fs, &RequestFile{
					Reader:        f,
					Seeker:        f,
					Closer:        f,
					Filename:      v.Filename,
					Headers:       hs,
					ContentLength: cl,
				})
			}

			r.Files[k] = fs
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

// RequestParams is an HTTP request param map.
type RequestParams map[string][]string

// First returns the first value associated with the key. It returns "" if there
// are no values associated with the key.
func (rps RequestParams) First(key string) string {
	if vs := rps[key]; len(vs) > 0 {
		return vs[0]
	}

	return ""
}

// Bool returns a `bool` by parsing the first value associated with the key. It
// returns (false, nil) if there are no values associated with the key.
func (rps RequestParams) Bool(key string) (bool, error) {
	if v := rps.First(key); v != "" {
		return strconv.ParseBool(v)
	}

	return false, nil
}

// Bools returns a `bool` slice by parsing the values associated with the key.
// It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Bools(key string) ([]bool, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	bs := make([]bool, 0, len(rps[key]))
	for _, v := range rps[key] {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}

		bs = append(bs, b)
	}

	return bs, nil
}

// Int returns an `int` by parsing the first value associated with the key. It
// returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Int(key string) (int, error) {
	if v := rps.First(key); v != "" {
		i64, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			return 0, err
		}

		return int(i64), nil
	}

	return 0, nil
}

// Ints returns an `int` slice by parsing the values associated with the key. It
// returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Ints(key string) ([]int, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	is := make([]int, 0, len(rps[key]))
	for _, v := range rps[key] {
		i64, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			return nil, err
		}

		is = append(is, int(i64))
	}

	return is, nil
}

// Int8 returns an `int8` by parsing the first value associated with the key. It
// returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Int8(key string) (int8, error) {
	if v := rps.First(key); v != "" {
		i64, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return 0, err
		}

		return int8(i64), nil
	}

	return 0, nil
}

// Int8s returns an `int8` slice by parsing the values associated with the key.
// It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Int8s(key string) ([]int8, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	i8s := make([]int8, 0, len(rps[key]))
	for _, v := range rps[key] {
		i64, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return nil, err
		}

		i8s = append(i8s, int8(i64))
	}

	return i8s, nil
}

// Int16 returns an `int16` by parsing the first value associated with the key.
// It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Int16(key string) (int16, error) {
	if v := rps.First(key); v != "" {
		i64, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return 0, err
		}

		return int16(i64), nil
	}

	return 0, nil
}

// Int16s returns an `int16s` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Int16s(key string) ([]int16, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	i16s := make([]int16, 0, len(rps[key]))
	for _, v := range rps[key] {
		i64, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return nil, err
		}

		i16s = append(i16s, int16(i64))
	}

	return i16s, nil
}

// Int32 returns an `int32` by parsing the first value associated with the key.
// It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Int32(key string) (int32, error) {
	if v := rps.First(key); v != "" {
		i64, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return 0, err
		}

		return int32(i64), nil
	}

	return 0, nil
}

// Int32s returns an `int32s` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Int32s(key string) ([]int32, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	i32s := make([]int32, 0, len(rps[key]))
	for _, v := range rps[key] {
		i64, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, err
		}

		i32s = append(i32s, int32(i64))
	}

	return i32s, nil
}

// Int64 returns an `int64` by parsing the first value associated with the key.
// It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Int64(key string) (int64, error) {
	if v := rps.First(key); v != "" {
		return strconv.ParseInt(v, 10, 64)
	}

	return 0, nil
}

// Int64s returns an `int64s` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Int64s(key string) ([]int64, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	i64s := make([]int64, 0, len(rps[key]))
	for _, v := range rps[key] {
		i64, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}

		i64s = append(i64s, i64)
	}

	return i64s, nil
}

// Uint returns an `uint` by parsing the first value associated with the key. It
// returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Uint(key string) (uint, error) {
	if v := rps.First(key); v != "" {
		ui64, err := strconv.ParseUint(v, 10, 0)
		if err != nil {
			return 0, err
		}

		return uint(ui64), nil
	}

	return 0, nil
}

// Uints returns an `uint` slice by parsing the values associated with the key.
// It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Uints(key string) ([]uint, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	is := make([]uint, 0, len(rps[key]))
	for _, v := range rps[key] {
		ui64, err := strconv.ParseUint(v, 10, 0)
		if err != nil {
			return nil, err
		}

		is = append(is, uint(ui64))
	}

	return is, nil
}

// Uint8 returns an `uint8` by parsing the first value associated with the key. It
// returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Uint8(key string) (uint8, error) {
	if v := rps.First(key); v != "" {
		ui64, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return 0, err
		}

		return uint8(ui64), nil
	}

	return 0, nil
}

// Uint8s returns an `uint8` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Uint8s(key string) ([]uint8, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	ui8s := make([]uint8, 0, len(rps[key]))
	for _, v := range rps[key] {
		ui64, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return nil, err
		}

		ui8s = append(ui8s, uint8(ui64))
	}

	return ui8s, nil
}

// Uint16 returns an `uint16` by parsing the first value associated with the
// key. It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Uint16(key string) (uint16, error) {
	if v := rps.First(key); v != "" {
		ui64, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return 0, err
		}

		return uint16(ui64), nil
	}

	return 0, nil
}

// Uint16s returns an `uint16s` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Uint16s(key string) ([]uint16, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	ui16s := make([]uint16, 0, len(rps[key]))
	for _, v := range rps[key] {
		ui64, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return nil, err
		}

		ui16s = append(ui16s, uint16(ui64))
	}

	return ui16s, nil
}

// Uint32 returns an `uint32` by parsing the first value associated with the
// key. It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Uint32(key string) (uint32, error) {
	if v := rps.First(key); v != "" {
		ui64, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, err
		}

		return uint32(ui64), nil
	}

	return 0, nil
}

// Uint32s returns an `uint32s` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Uint32s(key string) ([]uint32, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	ui32s := make([]uint32, 0, len(rps[key]))
	for _, v := range rps[key] {
		ui64, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, err
		}

		ui32s = append(ui32s, uint32(ui64))
	}

	return ui32s, nil
}

// Uint64 returns an `uint64` by parsing the first value associated with the
// key. It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Uint64(key string) (uint64, error) {
	if v := rps.First(key); v != "" {
		return strconv.ParseUint(v, 10, 64)
	}

	return 0, nil
}

// Uint64s returns an `uint64s` slice by parsing the values associated with the
// key. It returns (nil, nil) if there are no values associated with the key.
func (rps RequestParams) Uint64s(key string) ([]uint64, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	ui64s := make([]uint64, 0, len(rps[key]))
	for _, v := range rps[key] {
		ui64, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}

		ui64s = append(ui64s, ui64)
	}

	return ui64s, nil
}

// Float32 returns an `float32` by parsing the first value associated with the
// key. It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Float32(key string) (float32, error) {
	if v := rps.First(key); v != "" {
		f64, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return 0, err
		}

		return float32(f64), nil
	}

	return 0, nil
}

// Float32s returns an `float32s` slice by parsing the values associated with
// the key. It returns (nil, nil) if there are no values associated with the
// key.
func (rps RequestParams) Float32s(key string) ([]float32, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	f32s := make([]float32, 0, len(rps[key]))
	for _, v := range rps[key] {
		f64, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, err
		}

		f32s = append(f32s, float32(f64))
	}

	return f32s, nil
}

// Float64 returns an `float64` by parsing the first value associated with the
// key. It returns (0, nil) if there are no values associated with the key.
func (rps RequestParams) Float64(key string) (float64, error) {
	if v := rps.First(key); v != "" {
		return strconv.ParseFloat(v, 64)
	}

	return 0, nil
}

// Float64s returns an `float64s` slice by parsing the values associated with
// the key. It returns (nil, nil) if there are no values associated with the
// key.
func (rps RequestParams) Float64s(key string) ([]float64, error) {
	if len(rps[key]) == 0 {
		return nil, nil
	}

	f64s := make([]float64, 0, len(rps[key]))
	for _, v := range rps[key] {
		f64, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}

		f64s = append(f64s, f64)
	}

	return f64s, nil
}

// RequestFiles is an HTTP request file map.
type RequestFiles map[string][]*RequestFile

// First returns the first value associated with the key. It returns nil if
// there are no values associated with the key.
func (rfs RequestFiles) First(key string) *RequestFile {
	if vs := rfs[key]; len(vs) > 0 {
		return vs[0]
	}

	return nil
}

// RequestFile is an HTTP request file.
type RequestFile struct {
	io.Reader
	io.Seeker
	io.Closer

	Filename      string
	Headers       Headers
	ContentLength int64
}

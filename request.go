package air

import (
	"bytes"
	"io"
	"mime/multipart"

	"github.com/valyala/fasthttp"
)

// Request for HTTP request.
type Request struct {
	fastCtx *fasthttp.RequestCtx
	Header  Header
	URI     URI
	Logger  Logger
}

// NewRequest returns `Request` instance.
func NewRequest(c *fasthttp.RequestCtx, l Logger) *Request {
	return &Request{
		fastCtx: c,
		URI:     URI{fastURI: c.URI()},
		Header:  &fastRequestHeader{RequestHeader: &c.Request.Header},
		Logger:  l,
	}
}

// IsTLS returns true if HTTP connection is TLS otherwise false.
func (r *Request) IsTLS() bool {
	return r.fastCtx.IsTLS()
}

// Scheme returns the HTTP protocol scheme, `http` or `https`.
func (r *Request) Scheme() string {
	return string(r.fastCtx.Request.URI().Scheme())
}

// Host returns HTTP request host. Per RFC 2616, this is either the value of
// the `Host` header or the host name given in the URI itself.
func (r *Request) Host() string {
	return string(r.fastCtx.Request.Host())
}

// Referer returns the referring URI, if sent in the request.
func (r *Request) Referer() string {
	return string(r.fastCtx.Request.Header.Referer())
}

// ContentLength returns the size of request's body.
func (r *Request) ContentLength() int64 {
	return int64(r.fastCtx.Request.Header.ContentLength())
}

// UserAgent returns the client's `User-Agent`.
func (r *Request) UserAgent() string {
	return string(r.fastCtx.UserAgent())
}

// RemoteAddr returns the client's network address.
func (r *Request) RemoteAddr() string {
	return r.fastCtx.RemoteAddr().String()
}

// Method returns the request's HTTP function.
func (r *Request) Method() string {
	return string(r.fastCtx.Method())
}

// SetMethod sets the HTTP method of the request.
func (r *Request) SetMethod(method string) {
	r.fastCtx.Request.Header.SetMethodBytes([]byte(method))
}

// RequestURI returns the unmodified `Request-URI` sent by the client.
func (r *Request) RequestURI() string {
	return string(r.fastCtx.Request.RequestURI())
}

// SetURI sets the URI of the request.
func (r *Request) SetURI(uri string) {
	r.fastCtx.Request.Header.SetRequestURI(uri)
}

// Body returns request's body.
func (r *Request) Body() io.Reader {
	return bytes.NewBuffer(r.fastCtx.Request.Body())
}

// Body sets request's body.
func (r *Request) SetBody(reader io.Reader) {
	r.fastCtx.Request.SetBodyStream(reader, 0)
}

// FormValue returns the form field value for the provided name.
func (r *Request) FormValue(name string) string {
	return string(r.fastCtx.FormValue(name))
}

// FormParams returns the form parameters.
func (r *Request) FormParams() (params map[string][]string) {
	params = make(map[string][]string)
	mf, err := r.fastCtx.Request.MultipartForm()

	if err == fasthttp.ErrNoMultipartForm {
		r.fastCtx.PostArgs().VisitAll(func(k, v []byte) {
			key := string(k)
			if _, ok := params[key]; ok {
				params[key] = append(params[key], string(v))
			} else {
				params[string(k)] = []string{string(v)}
			}
		})
	} else if err == nil {
		for k, v := range mf.Value {
			if len(v) > 0 {
				params[k] = v
			}
		}
	}

	return
}

// FormFile returns the multipart form file for the provided name.
func (r *Request) FormFile(name string) (*multipart.FileHeader, error) {
	return r.fastCtx.FormFile(name)
}

// MultipartForm returns the multipart form.
func (r *Request) MultipartForm() (*multipart.Form, error) {
	return r.fastCtx.MultipartForm()
}

// Cookie returns the named cookie provided in the request.
func (r *Request) Cookie(name string) (Cookie, error) {
	c := new(fasthttp.Cookie)
	b := r.fastCtx.Request.Header.Cookie(name)
	if b == nil {
		return Cookie{}, ErrCookieNotFound
	}
	c.SetKey(name)
	c.SetValueBytes(b)
	return Cookie{c}, nil
}

// Cookies returns the HTTP cookies sent with the request.
func (r *Request) Cookies() []Cookie {
	cookies := []Cookie{}
	r.fastCtx.Request.Header.VisitAllCookie(func(name, value []byte) {
		c := new(fasthttp.Cookie)
		c.SetKeyBytes(name)
		c.SetValueBytes(value)
		cookies = append(cookies, Cookie{c})
	})
	return cookies
}

func (r *Request) reset(c *fasthttp.RequestCtx, h Header, u URI) {
	r.fastCtx = c
	r.Header = h
	r.URI = u
}

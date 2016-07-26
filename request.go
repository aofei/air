package air

import (
	"bytes"
	"io"
	"mime/multipart"

	"github.com/valyala/fasthttp"
)

type (
	// Request defines the interface for HTTP request.
	Request interface {
		// IsTLS returns true if HTTP connection is TLS otherwise false.
		IsTLS() bool

		// Scheme returns the HTTP protocol scheme, `http` or `https`.
		Scheme() string

		// Host returns HTTP request host. Per RFC 2616, this is either the value of
		// the `Host` header or the host name given in the URI itself.
		Host() string

		// RequestURI returns the unmodified `Request-URI` sent by the client.
		RequestURI() string

		// SetURI sets the URI of the request.
		SetURI(string)

		// URI returns `fasthttp.URI`.
		URI() URI

		// Header returns `fasthttp.Header`.
		Header() Header

		// Referer returns the referring URI, if sent in the request.
		Referer() string

		// Protocol returns the protocol version string of the HTTP request.
		// Protocol() string

		// ProtocolMajor returns the major protocol version of the HTTP request.
		// ProtocolMajor() int

		// ProtocolMinor returns the minor protocol version of the HTTP request.
		// ProtocolMinor() int

		// ContentLength returns the size of request's body.
		ContentLength() int64

		// UserAgent returns the client's `User-Agent`.
		UserAgent() string

		// RemoteAddress returns the client's network address.
		RemoteAddress() string

		// Method returns the request's HTTP function.
		Method() string

		// SetMethod sets the HTTP method of the request.
		SetMethod(string)

		// Body returns request's body.
		Body() io.Reader

		// Body sets request's body.
		SetBody(io.Reader)

		// FormValue returns the form field value for the provided name.
		FormValue(string) string

		// FormParams returns the form parameters.
		FormParams() map[string][]string

		// FormFile returns the multipart form file for the provided name.
		FormFile(string) (*multipart.FileHeader, error)

		// MultipartForm returns the multipart form.
		MultipartForm() (*multipart.Form, error)

		// Cookie returns the named cookie provided in the request.
		Cookie(string) (Cookie, error)

		// Cookies returns the HTTP cookies sent with the request.
		Cookies() []Cookie
	}

	// FastRequest implements `Request`.
	FastRequest struct {
		*fasthttp.RequestCtx
		header Header
		uri    URI
		logger Logger
	}
)

// NewRequest returns `FastRequest` instance.
func NewRequest(c *fasthttp.RequestCtx, l Logger) *FastRequest {
	return &FastRequest{
		RequestCtx: c,
		uri:        &FastURI{URI: c.URI()},
		header:     &fastRequestHeader{RequestHeader: &c.Request.Header},
		logger:     l,
	}
}

// IsTLS implements `Request#TLS` function.
func (r *FastRequest) IsTLS() bool {
	return r.RequestCtx.IsTLS()
}

// Scheme implements `Request#Scheme` function.
func (r *FastRequest) Scheme() string {
	return string(r.RequestCtx.URI().Scheme())
}

// Host implements `Request#Host` function.
func (r *FastRequest) Host() string {
	return string(r.RequestCtx.Host())
}

// URI implements `Request#URI` function.
func (r *FastRequest) URI() URI {
	return r.uri
}

// Header implements `Request#Header` function.
func (r *FastRequest) Header() Header {
	return r.header
}

// Referer implements `Request#Referer` function.
func (r *FastRequest) Referer() string {
	return string(r.Request.Header.Referer())
}

// ContentLength implements `Request#ContentLength` function.
func (r *FastRequest) ContentLength() int64 {
	return int64(r.Request.Header.ContentLength())
}

// UserAgent implements `Request#UserAgent` function.
func (r *FastRequest) UserAgent() string {
	return string(r.RequestCtx.UserAgent())
}

// RemoteAddress implements `Request#RemoteAddress` function.
func (r *FastRequest) RemoteAddress() string {
	return r.RemoteAddr().String()
}

// Method implements `Request#Method` function.
func (r *FastRequest) Method() string {
	return string(r.RequestCtx.Method())
}

// SetMethod implements `Request#SetMethod` function.
func (r *FastRequest) SetMethod(method string) {
	r.Request.Header.SetMethodBytes([]byte(method))
}

// RequestURI implements `Request#RequestURI` function.
func (r *FastRequest) RequestURI() string {
	return string(r.Request.RequestURI())
}

// SetURI implements `Request#SetURI` function.
func (r *FastRequest) SetURI(uri string) {
	r.Request.Header.SetRequestURI(uri)
}

// Body implements `Request#Body` function.
func (r *FastRequest) Body() io.Reader {
	return bytes.NewBuffer(r.Request.Body())
}

// SetBody implements `Request#SetBody` function.
func (r *FastRequest) SetBody(reader io.Reader) {
	r.Request.SetBodyStream(reader, 0)
}

// FormValue implements `Request#FormValue` function.
func (r *FastRequest) FormValue(name string) string {
	return string(r.RequestCtx.FormValue(name))
}

// FormParams implements `Request#FormParams` function.
func (r *FastRequest) FormParams() (params map[string][]string) {
	params = make(map[string][]string)
	mf, err := r.RequestCtx.MultipartForm()

	if err == fasthttp.ErrNoMultipartForm {
		r.PostArgs().VisitAll(func(k, v []byte) {
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

// FormFile implements `Request#FormFile` function.
func (r *FastRequest) FormFile(name string) (*multipart.FileHeader, error) {
	return r.RequestCtx.FormFile(name)
}

// MultipartForm implements `Request#MultipartForm` function.
func (r *FastRequest) MultipartForm() (*multipart.Form, error) {
	return r.RequestCtx.MultipartForm()
}

// Cookie implements `Request#Cookie` function.
func (r *FastRequest) Cookie(name string) (Cookie, error) {
	c := new(fasthttp.Cookie)
	b := r.Request.Header.Cookie(name)
	if b == nil {
		return nil, ErrCookieNotFound
	}
	c.SetKey(name)
	c.SetValueBytes(b)
	return &fastCookie{c}, nil
}

// Cookies implements `Request#Cookies` function.
func (r *FastRequest) Cookies() []Cookie {
	cookies := []Cookie{}
	r.Request.Header.VisitAllCookie(func(name, value []byte) {
		c := new(fasthttp.Cookie)
		c.SetKeyBytes(name)
		c.SetValueBytes(value)
		cookies = append(cookies, &fastCookie{c})
	})
	return cookies
}

func (r *FastRequest) reset(c *fasthttp.RequestCtx, h Header, u URI) {
	r.RequestCtx = c
	r.header = h
	r.uri = u
}

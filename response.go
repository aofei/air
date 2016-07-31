package air

import (
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
)

type (
	// Response for HTTP response.
	Response struct {
		fastCtx   *fasthttp.RequestCtx
		Header    ResponseHeader
		Status    int
		Size      int64
		Committed bool
		Writer    io.Writer
		Logger    Logger
	}
)

// NewResponse returns `Response` instance.
func NewResponse(c *fasthttp.RequestCtx, l Logger) *Response {
	return &Response{
		fastCtx: c,
		Header:  ResponseHeader{fastResponseHeader: &c.Response.Header},
		Writer:  c,
		Logger:  l,
	}
}

// WriteHeader sends an HTTP response header with status code.
func (r *Response) WriteHeader(code int) {
	if r.Committed {
		r.Logger.Warn("Response Already Committed")
		return
	}
	r.Status = code
	r.fastCtx.SetStatusCode(code)
	r.Committed = true
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(b []byte) (n int, err error) {
	if !r.Committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.Writer.Write(b)
	r.Size += int64(n)
	return
}

// SetCookie adds a `Set-Cookie` header in HTTP response.
func (r *Response) SetCookie(c Cookie) {
	cookie := new(fasthttp.Cookie)
	cookie.SetKey(c.Name())
	cookie.SetValue(c.Value())
	cookie.SetPath(c.Path())
	cookie.SetDomain(c.Domain())
	cookie.SetExpire(c.Expires())
	cookie.SetSecure(c.Secure())
	cookie.SetHTTPOnly(c.HTTPOnly())
	r.fastCtx.Response.Header.SetCookie(cookie)
}

func (r *Response) reset(c *fasthttp.RequestCtx, h ResponseHeader) {
	r.fastCtx = c
	r.Header = h
	r.Status = http.StatusOK
	r.Size = 0
	r.Committed = false
	r.Writer = c
}

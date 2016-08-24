package air

import (
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
)

// Response represents the current HTTP response.
type Response struct {
	fastCtx *fasthttp.RequestCtx
	air     *Air

	Header    *ResponseHeader
	Size      int64
	Committed bool
	Writer    io.Writer
}

// newResponse returns a new instance of `Response`.
func newResponse(a *Air) *Response {
	return &Response{
		air: a,
	}
}

// WriteHeader sends an HTTP response header with status code.
func (r *Response) WriteHeader(code int) {
	if r.Committed {
		r.air.Logger.Warn("Response Already Committed")
		return
	}
	r.fastCtx.SetStatusCode(code)
	r.Committed = true
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(b []byte) (int, error) {
	if !r.Committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.Writer.Write(b)
	r.Size += int64(n)
	return n, err
}

// SetCookie adds a "Set-Cookie" header in HTTP response.
func (r *Response) SetCookie(c Cookie) {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(c.Name())
	cookie.SetValue(c.Value())
	cookie.SetPath(c.Path())
	cookie.SetDomain(c.Domain())
	cookie.SetExpire(c.Expires())
	cookie.SetSecure(c.Secure())
	cookie.SetHTTPOnly(c.HTTPOnly())
	r.fastCtx.Response.Header.SetCookie(cookie)
}

// reset resets the instance of `Response`.
func (r *Response) reset() {
	r.fastCtx = nil
	r.Header = nil
	r.Size = 0
	r.Committed = false
	r.Writer = nil
}

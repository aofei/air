package air

import (
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
)

// Response represents the current HTTP response.
type Response struct {
	fastCtx *fasthttp.RequestCtx

	Header    *ResponseHeader
	Status    int
	Size      int64
	Committed bool
	Writer    io.Writer
	Logger    Logger
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

// SetCookie adds a "Set-Cookie" header in HTTP response.
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

// reset resets the instance of `Response`.
func (r *Response) reset(c *fasthttp.RequestCtx, h *ResponseHeader) {
	r.fastCtx = c
	r.Header = h
	r.Status = http.StatusOK
	r.Size = 0
	r.Committed = false
	r.Writer = c
}

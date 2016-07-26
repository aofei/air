package air

import (
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
)

type (
	// Response defines the interface for HTTP response.
	Response interface {
		// Header returns `fasthttp.Header`
		Header() Header

		// WriteHeader sends an HTTP response header with status code.
		WriteHeader(int)

		// Write writes the data to the connection as part of an HTTP reply.
		Write(b []byte) (int, error)

		// SetCookie adds a `Set-Cookie` header in HTTP response.
		SetCookie(Cookie)

		// Status returns the HTTP response status.
		Status() int

		// Size returns the number of bytes written to HTTP response.
		Size() int64

		// Committed returns true if HTTP response header is written, otherwise false.
		Committed() bool

		// Write returns the HTTP response writer.
		Writer() io.Writer

		// SetWriter sets the HTTP response writer.
		SetWriter(io.Writer)
	}

	fastResponse struct {
		*fasthttp.RequestCtx
		header    Header
		status    int
		size      int64
		committed bool
		writer    io.Writer
		logger    Logger
	}
)

// NewResponse returns `fastResponse` instance.
func NewResponse(c *fasthttp.RequestCtx, l Logger) *fastResponse {
	return &fastResponse{
		RequestCtx: c,
		header:     &fastResponseHeader{ResponseHeader: &c.Response.Header},
		writer:     c,
		logger:     l,
	}
}

func (r *fastResponse) Header() Header {
	return r.header
}

func (r *fastResponse) WriteHeader(code int) {
	if r.committed {
		r.logger.Warn("response already committed")
		return
	}
	r.status = code
	r.SetStatusCode(code)
	r.committed = true
}

func (r *fastResponse) Write(b []byte) (n int, err error) {
	if !r.committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.writer.Write(b)
	r.size += int64(n)
	return
}

func (r *fastResponse) SetCookie(c Cookie) {
	cookie := new(fasthttp.Cookie)
	cookie.SetKey(c.Name())
	cookie.SetValue(c.Value())
	cookie.SetPath(c.Path())
	cookie.SetDomain(c.Domain())
	cookie.SetExpire(c.Expires())
	cookie.SetSecure(c.Secure())
	cookie.SetHTTPOnly(c.HTTPOnly())
	r.Response.Header.SetCookie(cookie)
}

func (r *fastResponse) Status() int {
	return r.status
}

func (r *fastResponse) Size() int64 {
	return r.size
}

func (r *fastResponse) Committed() bool {
	return r.committed
}

func (r *fastResponse) Writer() io.Writer {
	return r.writer
}

func (r *fastResponse) SetWriter(w io.Writer) {
	r.writer = w
}

func (r *fastResponse) reset(c *fasthttp.RequestCtx, h Header) {
	r.RequestCtx = c
	r.header = h
	r.status = http.StatusOK
	r.size = 0
	r.committed = false
	r.writer = c
}

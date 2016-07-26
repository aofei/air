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

	// FastResponse implements `Response`.
	FastResponse struct {
		*fasthttp.RequestCtx
		header    Header
		status    int
		size      int64
		committed bool
		writer    io.Writer
		logger    Logger
	}
)

// NewResponse returns `FastResponse` instance.
func NewResponse(c *fasthttp.RequestCtx, l Logger) *FastResponse {
	return &FastResponse{
		RequestCtx: c,
		header:     &fastResponseHeader{ResponseHeader: &c.Response.Header},
		writer:     c,
		logger:     l,
	}
}

// Header implements `Response#Header` function.
func (r *FastResponse) Header() Header {
	return r.header
}

// WriteHeader implements `Response#WriteHeader` function.
func (r *FastResponse) WriteHeader(code int) {
	if r.committed {
		r.logger.Warn("response already committed")
		return
	}
	r.status = code
	r.SetStatusCode(code)
	r.committed = true
}

// Write implements `Response#Write` function.
func (r *FastResponse) Write(b []byte) (n int, err error) {
	if !r.committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.writer.Write(b)
	r.size += int64(n)
	return
}

// SetCookie implements `Response#SetCookie` function.
func (r *FastResponse) SetCookie(c Cookie) {
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

// Status implements `Response#Status` function.
func (r *FastResponse) Status() int {
	return r.status
}

// Size implements `Response#Size` function.
func (r *FastResponse) Size() int64 {
	return r.size
}

// Committed implements `Response#Committed` function.
func (r *FastResponse) Committed() bool {
	return r.committed
}

// Writer implements `Response#Writer` function.
func (r *FastResponse) Writer() io.Writer {
	return r.writer
}

// SetWriter implements `Response#SetWriter` function.
func (r *FastResponse) SetWriter(w io.Writer) {
	r.writer = w
}

func (r *FastResponse) reset(c *fasthttp.RequestCtx, h Header) {
	r.RequestCtx = c
	r.header = h
	r.status = http.StatusOK
	r.size = 0
	r.committed = false
	r.writer = c
}

package air

import (
	"net/http"
	"net/url"
)

// Request represents the current HTTP request.
//
// It's embedded with the `http.Request`.
type Request struct {
	*http.Request

	context *Context

	URL *URL
}

// NewRequest returns a pointer of a new instance of the `Request`.
func NewRequest(c *Context) *Request {
	req := &Request{context: c}
	req.URL = NewURL(req)
	return req
}

// Bind binds the HTTP body of the req into the provided type i. The default `Binder` does it based
// on the "Content-Type" header.
func (req *Request) Bind(i interface{}) error {
	return req.context.Air.Binder.Bind(i, req)
}

// FormValues returns the form values.
func (req *Request) FormValues() url.Values {
	if req.Form == nil {
		req.ParseMultipartForm(32 << 20) // The maxMemory is 32 MB
	}
	return req.Form
}

// feed feeds the r into where it should be.
func (req *Request) feed(r *http.Request) {
	req.Request = r
	req.URL.feed(r.URL)
}

// reset resets all fields in the req.
func (req *Request) reset() {
	req.Request = nil
	req.URL.reset()
}

// MARK: Alias methods for the `Request#URL`.

// QueryValue is an alias for the `URL#QueryValue()` of the req.
func (req *Request) QueryValue(key string) string {
	return req.URL.QueryValue(key)
}

// QueryValues is an alias for the `URL#QueryValues()` of the req.
func (req *Request) QueryValues() url.Values {
	return req.URL.QueryValues()
}

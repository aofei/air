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
	r := &Request{context: c}
	r.URL = NewURL(r)
	return r
}

// Bind binds the HTTP body of the r into the provided type i. The default `Binder` does it based
// on the "Content-Type" header.
func (r *Request) Bind(i interface{}) error {
	return r.context.Air.Binder.Bind(i, r)
}

// FormValues returns the form values.
func (r *Request) FormValues() url.Values {
	if r.Form == nil {
		r.ParseMultipartForm(32 << 20) // The maxMemory is 32 MB
	}
	return r.Form
}

// HasFormValue reports whether the form values contains the form value for the provided key.
func (r *Request) HasFormValue(key string) bool {
	for k := range r.FormValues() {
		if k == key {
			return true
		}
	}
	return false
}

// feed feeds the req into where it should be.
func (r *Request) feed(req *http.Request) {
	r.Request = req
	r.URL.feed(req.URL)
}

// reset resets all fields in the r.
func (r *Request) reset() {
	r.Request = nil
	r.URL.reset()
}

// MARK: Alias methods for the `Request#URL`.

// QueryValue is an alias for the `URL#QueryValue()` of the r.
func (r *Request) QueryValue(key string) string {
	return r.URL.QueryValue(key)
}

// QueryValues is an alias for the `URL#QueryValues()` of the r.
func (r *Request) QueryValues() url.Values {
	return r.URL.QueryValues()
}

// HasQueryValue is an alias for the `URL#HasQueryValue()` of the r.
func (r *Request) HasQueryValue(key string) bool {
	return r.URL.HasQueryValue(key)
}

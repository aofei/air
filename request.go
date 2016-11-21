package air

import (
	"net/http"
	"net/url"
	"strings"
)

// Request represents the current HTTP request.
//
// It's embedded with `http.Request`.
type Request struct {
	*http.Request

	context *Context

	URL *URL
}

const defaultMemory = 32 << 20 // 32 MB

// newRequest returns a pointer of a new instance of `Request`.
func newRequest(c *Context) *Request {
	return &Request{
		context: c,
		URL:     newURL(),
	}
}

// Bind binds the HTTP body of the req into provided type i. The default binder does it based on
// "Content-Type" header.
func (req *Request) Bind(i interface{}) error {
	return req.context.Air.Binder.Bind(i, req)
}

// FormValues returns the form values.
func (req *Request) FormValues() (url.Values, error) {
	if strings.HasPrefix(req.Header.Get(HeaderContentType), MIMEMultipartForm) {
		if err := req.ParseMultipartForm(defaultMemory); err != nil {
			return nil, err
		}
	} else {
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
	}
	return req.Form, nil
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

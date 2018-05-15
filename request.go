package air

import (
	"io"
	"net/http"
)

// Request is an HTTP request.
type Request struct {
	Method        string
	URL           *URL
	Proto         string
	Headers       map[string]string
	ContentLength int64
	Body          io.Reader
	Cookies       []*Cookie
	Params        map[string]string
	Files         map[string]io.Reader
	RemoteAddr    string
	Values        map[string]interface{}

	request *http.Request
}

// newRequest returns a new instance of the `Request`.
func newRequest(r *http.Request) *Request {
	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	cookies := make([]*Cookie, 0, len(r.Header["Cookie"]))
	for _, line := range r.Header["Cookie"] {
		cookies = append(cookies, newCookie(line))
	}

	if r.Form == nil || r.MultipartForm == nil {
		r.ParseMultipartForm(32 << 20)
	}

	params := make(map[string]string, len(r.Form)+theRouter.maxParams)
	for k, v := range r.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	files := make(map[string]io.Reader, 0)
	if r.MultipartForm != nil {
		files = make(map[string]io.Reader, len(r.MultipartForm.File))
		for k, v := range r.MultipartForm.File {
			if len(v) > 0 {
				f, err := v[0].Open()
				if err == nil {
					files[k] = f
				}
			}
		}
	}

	return &Request{
		Method:        r.Method,
		URL:           newURL(r.URL),
		Proto:         r.Proto,
		Headers:       headers,
		ContentLength: r.ContentLength,
		Body:          r.Body,
		Cookies:       cookies,
		Params:        params,
		Files:         files,
		RemoteAddr:    r.RemoteAddr,
		Values:        map[string]interface{}{},

		request: r,
	}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return theBinder.bind(v, r)
}

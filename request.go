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
	PathParams    map[string]string
	QueryParams   map[string]string
	FormParams    map[string]string
	FormFiles     map[string]io.Reader
	Values        map[string]interface{}

	request *http.Request
}

// newRequest returns a new instance of the `Request`.
func newRequest(r *http.Request) *Request {
	headers := map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	cookies := []*Cookie{}
	for _, c := range r.Cookies() {
		cookies = append(cookies, newCookie(c))
	}

	queryParams := map[string]string{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	if r.Form == nil || r.MultipartForm == nil {
		r.ParseMultipartForm(32 << 20)
	}

	formParams := map[string]string{}
	for k, v := range r.Form {
		if len(v) > 0 {
			formParams[k] = v[0]
		}
	}

	formFiles := map[string]io.Reader{}
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.File {
			if len(v) > 0 {
				f, err := v[0].Open()
				if err == nil {
					formFiles[k] = f
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
		PathParams:    map[string]string{},
		QueryParams:   queryParams,
		FormParams:    formParams,
		FormFiles:     formFiles,
		Values:        map[string]interface{}{},

		request: r,
	}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return binderSingleton.bind(v, r)
}

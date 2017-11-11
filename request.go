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

	cs := r.Cookies()
	cookies := make([]*Cookie, 0, len(cs))
	for _, c := range cs {
		cookies = append(cookies, newCookie(c))
	}

	qps := r.URL.Query()
	queryParams := make(map[string]string, len(qps))
	for k, v := range qps {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	if r.Form == nil || r.MultipartForm == nil {
		r.ParseMultipartForm(32 << 20)
	}

	formParams := make(map[string]string, len(r.Form))
	for k, v := range r.Form {
		if len(v) > 0 {
			formParams[k] = v[0]
		}
	}

	formFiles := make(map[string]io.Reader, 0)
	if mf := r.MultipartForm; mf != nil {
		formFiles = make(map[string]io.Reader, len(mf.File))
		for k, v := range mf.File {
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
		PathParams:    make(map[string]string, theRouter.maxParams),
		QueryParams:   queryParams,
		FormParams:    formParams,
		FormFiles:     formFiles,
		RemoteAddr:    r.RemoteAddr,
		Values:        map[string]interface{}{},

		request: r,
	}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return theBinder.bind(v, r)
}

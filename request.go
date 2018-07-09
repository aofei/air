package air

import (
	"io"
	"mime/multipart"
)

// Request is an HTTP request.
type Request struct {
	Method        string
	URL           *URL
	Proto         string
	Headers       map[string]string
	Body          io.Reader
	ContentLength int64
	Cookies       []*Cookie
	Params        map[string]string
	Files         map[string]multipart.File
	RemoteAddr    string
	Values        map[string]interface{}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return theBinder.bind(v, r)
}

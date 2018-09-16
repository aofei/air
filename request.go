package air

import (
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
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
	ClientIP      net.IP
	Values        map[string]interface{}

	httpRequest     *http.Request
	localizedString func(string) string
}

// ParseCookies parses the cookies sent with the r into the `r.Cookies`.
//
// It must be called manually when the `ParseRequestCookiesManually` is true.
func (r *Request) ParseCookies() {
	for _, line := range r.httpRequest.Header["Cookie"] {
		ps := strings.Split(strings.TrimSpace(line), ";")
		if len(ps) == 1 && ps[0] == "" {
			continue
		}

		for i := 0; i < len(ps); i++ {
			ps[i] = strings.TrimSpace(ps[i])
			if len(ps[i]) == 0 {
				continue
			}

			n, v := ps[i], ""
			if i := strings.Index(n, "="); i >= 0 {
				n, v = n[:i], n[i+1:]
			}

			if !validCookieName(n) {
				continue
			}

			if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}

			if !validCookieValue(v) {
				continue
			}

			r.Cookies = append(r.Cookies, &Cookie{
				Name:  n,
				Value: v,
			})
		}
	}
}

// ParseParams parses the params sent with the r into the `r.Params`.
//
// It must be called manually when the `ParseRequestParamsManually` is true.
func (r *Request) ParseParams() {
	if r.httpRequest.Form == nil {
		r.httpRequest.ParseForm()
	}

	for k, v := range r.httpRequest.Form {
		if len(v) > 0 {
			r.Params[k] = v[0]
		}
	}
}

// ParseFiles parses the files sent with the r into the `r.Files`.
//
// It must be called manually when the `ParseRequestFilesManually` is true.
func (r *Request) ParseFiles() {
	if r.httpRequest.MultipartForm == nil {
		r.httpRequest.ParseMultipartForm(32 << 20)
	}

	if r.httpRequest.MultipartForm != nil {
		for k, v := range r.httpRequest.MultipartForm.Value {
			if len(v) > 0 {
				r.Params[k] = v[0]
			}
		}

		for k, v := range r.httpRequest.MultipartForm.File {
			if len(v) > 0 {
				if f, err := v[0].Open(); err == nil {
					r.Files[k] = f
				}
			}
		}
	}
}

// Bind binds the r into the v.
func (r *Request) Bind(v interface{}) error {
	return theBinder.bind(v, r)
}

// LocalizedString returns localized string for the provided key.
//
// It only works if the `I18nEnabled` is true.
func (r *Request) LocalizedString(key string) string {
	if r.localizedString != nil {
		return r.localizedString(key)
	}

	return key
}

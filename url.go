package air

import (
	"bytes"
	"net/url"
)

// URL is an HTTP URL.
type URL struct {
	Scheme string
	Host   string
	Path   string
	Query  string
}

// newURL returns a new instance of the `URL`.
func newURL(u *url.URL) *URL {
	return &URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.EscapedPath(),
		Query:  u.RawQuery,
	}
}

// String returns the serialization string of the u.
func (u *URL) String() string {
	buf := bytes.Buffer{}
	if u.Scheme != "" {
		buf.WriteString(u.Scheme)
		buf.WriteByte(':')
	}
	if u.Scheme != "" || u.Host != "" {
		buf.WriteString("//")
		if u.Host != "" {
			buf.WriteString(u.Host)
		}
	}
	if u.Path != "" && u.Path[0] != '/' && u.Host != "" {
		buf.WriteByte('/')
	}
	buf.WriteString(u.Path)
	if u.Query != "" {
		buf.WriteByte('?')
		buf.WriteString(u.Query)
	}
	return buf.String()
}

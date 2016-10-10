package air

import "github.com/valyala/fasthttp"

// URI represents the current HTTP URI.
type URI struct {
	fastURI *fasthttp.URI
}

// newURI returns a new instance of `URI`.
func newURI() *URI {
	return &URI{}
}

// FullURI returns the full request URI.
func (u *URI) FullURI() string {
	return string(u.fastURI.FullURI())
}

// RequestURI returns the request URI.
func (u *URI) RequestURI() string {
	return string(u.fastURI.RequestURI())
}

// Path returns the request URI path. The returned path is
// always urldecoded and normalized.
func (u *URI) Path() string {
	return string(u.fastURI.Path())
}

// PathOriginal returns the original request URI path. The
// returned value is valid until the next URI method call.
func (u *URI) PathOriginal() string {
	return string(u.fastURI.PathOriginal())
}

// SetPath sets the request URI path.
func (u *URI) SetPath(path string) {
	u.fastURI.SetPath(path)
}

// QueryParam returns the query param for the provided name.
func (u *URI) QueryParam(name string) string {
	return string(u.fastURI.QueryArgs().Peek(name))
}

// QueryParams returns the query parameters as map.
func (u *URI) QueryParams() map[string][]string {
	params := make(map[string][]string)
	u.fastURI.QueryArgs().VisitAll(func(k, v []byte) {
		_, ok := params[string(k)]
		if !ok {
			params[string(k)] = make([]string, 0)
		}
		params[string(k)] = append(params[string(k)], string(v))
	})
	return params
}

// QueryString returns the URI query string.
func (u *URI) QueryString() string {
	return string(u.fastURI.QueryString())
}

// reset resets the instance of `URI`.
func (u *URI) reset() {
	u.fastURI = nil
}

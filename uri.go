package air

import "github.com/valyala/fasthttp"

// URI for HTTP request URI.
type URI struct {
	fastURI *fasthttp.URI
}

// FullURI returns the full request URI.
func (u *URI) FullURI() string {
	return string(u.fastURI.FullURI())
}

// Path returns the request URI path.
func (u *URI) Path() string {
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

// QueryParam returns the query parameters as map.
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

// reset resets the `URI` instance.
func (u *URI) reset(uri *fasthttp.URI) {
	u.fastURI = uri
}

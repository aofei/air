package air

import "github.com/valyala/fasthttp"

type (
	// URI defines the interface for HTTP request uri.
	URI interface {
		// FullURI returns full uri in the form {Scheme}://{Host}{RequestURI}#{Hash}.
		FullURI() string

		// Path returns the request URI path.
		Path() string

		// SetPath sets the request URI path.
		SetPath(string)

		// QueryParam returns the query param for the provided name.
		QueryParam(string) string

		// QueryParam returns the query parameters as map.
		QueryParams() map[string][]string

		// QueryString returns the URI query string.
		QueryString() string
	}

	fastURI struct {
		*fasthttp.URI
	}
)

func (u *fastURI) FullURI() string {
	return string(u.URI.FullURI())
}

func (u *fastURI) Path() string {
	return string(u.URI.PathOriginal())
}

func (u *fastURI) SetPath(path string) {
	u.URI.SetPath(path)
}

func (u *fastURI) QueryParam(name string) string {
	return string(u.URI.QueryArgs().Peek(name))
}

func (u *fastURI) QueryParams() map[string][]string {
	params := make(map[string][]string)
	u.URI.QueryArgs().VisitAll(func(k, v []byte) {
		_, ok := params[string(k)]
		if !ok {
			params[string(k)] = make([]string, 0)
		}
		params[string(k)] = append(params[string(k)], string(v))
	})
	return params
}

func (u *fastURI) QueryString() string {
	return string(u.URI.QueryString())
}

func (u *fastURI) reset(uri *fasthttp.URI) {
	u.URI = uri
}

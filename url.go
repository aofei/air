package air

import "github.com/valyala/fasthttp"

type (
	// URL defines the interface for HTTP request url.
	URL interface {
		// Path returns the request URL path.
		Path() string

		// SetPath sets the request URL path.
		SetPath(string)

		// QueryParam returns the query param for the provided name.
		QueryParam(string) string

		// QueryParam returns the query parameters as map.
		QueryParams() map[string][]string

		// QueryString returns the URL query string.
		QueryString() string
	}

	// FastURL implements `URL`.
	FastURL struct {
		*fasthttp.URI
	}
)

// Path implements `URL#Path` function.
func (u *FastURL) Path() string {
	return string(u.URI.PathOriginal())
}

// SetPath implements `URL#SetPath` function.
func (u *FastURL) SetPath(path string) {
	u.URI.SetPath(path)
}

// QueryParam implements `URL#QueryParam` function.
func (u *FastURL) QueryParam(name string) string {
	return string(u.QueryArgs().Peek(name))
}

// QueryParams implements `URL#QueryParams` function.
func (u *FastURL) QueryParams() (params map[string][]string) {
	params = make(map[string][]string)
	u.QueryArgs().VisitAll(func(k, v []byte) {
		_, ok := params[string(k)]
		if !ok {
			params[string(k)] = make([]string, 0)
		}
		params[string(k)] = append(params[string(k)], string(v))
	})
	return
}

// QueryString implements `URL#QueryString` function.
func (u *FastURL) QueryString() string {
	return string(u.URI.QueryString())
}

func (u *FastURL) reset(uri *fasthttp.URI) {
	u.URI = uri
}

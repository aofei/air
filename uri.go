package air

import "github.com/valyala/fasthttp"

type (
	// URI defines the interface for HTTP request uri.
	URI interface {
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

	// FastURI implements `URI`.
	FastURI struct {
		*fasthttp.URI
	}
)

// Path implements `URI#Path` function.
func (u *FastURI) Path() string {
	return string(u.URI.PathOriginal())
}

// SetPath implements `URI#SetPath` function.
func (u *FastURI) SetPath(path string) {
	u.URI.SetPath(path)
}

// QueryParam implements `URI#QueryParam` function.
func (u *FastURI) QueryParam(name string) string {
	return string(u.QueryArgs().Peek(name))
}

// QueryParams implements `URI#QueryParams` function.
func (u *FastURI) QueryParams() (params map[string][]string) {
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

// QueryString implements `URI#QueryString` function.
func (u *FastURI) QueryString() string {
	return string(u.URI.QueryString())
}

func (u *FastURI) reset(uri *fasthttp.URI) {
	u.URI = uri
}

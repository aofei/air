package air

import "net/url"

// URL represents the HTTP URL of the current HTTP request.
//
// It's embedded with `url.URL`.
type URL struct {
	*url.URL

	queryValues url.Values
}

// newURL returns a pointer of a new instance of `URL`.
func newURL() *URL {
	return &URL{}
}

// QueryValue returns the query value for the provided key.
func (url *URL) QueryValue(key string) string {
	if url.queryValues == nil {
		url.queryValues = url.Query()
	}
	return url.queryValues.Get(key)
}

// QueryValues returns the query values.
func (url *URL) QueryValues() url.Values {
	if url.queryValues == nil {
		url.queryValues = url.Query()
	}
	return url.queryValues
}

// reset resets all fields in the url.
func (url *URL) reset() {
	url.URL = nil
	url.queryValues = nil
}

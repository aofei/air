package air

import "net/url"

// URL represents the HTTP URL of the current HTTP request.
//
// It's embedded with the `url.URL`.
type URL struct {
	*url.URL

	request *Request

	queryValues url.Values
}

// NewURL returns a pointer of a new instance of the `URL`.
func NewURL(req *Request) *URL {
	return &URL{
		request: req,
	}
}

// QueryValue returns the query value in the url for the provided key.
func (url *URL) QueryValue(key string) string {
	return url.QueryValues().Get(key)
}

// QueryValues returns the query values in the url.
func (url *URL) QueryValues() url.Values {
	if url.queryValues == nil {
		url.queryValues = url.Query()
	}
	return url.queryValues
}

// feed feeds the u into where it should be.
func (url *URL) feed(u *url.URL) {
	url.URL = u
}

// reset resets all fields in the url.
func (url *URL) reset() {
	url.URL = nil
	url.queryValues = nil
}

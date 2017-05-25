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
func NewURL(r *Request) *URL {
	return &URL{
		request: r,
	}
}

// QueryValue returns the query value in the u for the provided key.
func (u *URL) QueryValue(key string) string {
	return u.QueryValues().Get(key)
}

// QueryValues returns the query values in the u.
func (u *URL) QueryValues() url.Values {
	if u.queryValues == nil {
		u.queryValues = u.Query()
	}
	return u.queryValues
}

// HasQueryValue reports whether the query values contains the query value for the provided key.
func (u *URL) HasQueryValue(key string) bool {
	for k, _ := range u.QueryValues() {
		if k == key {
			return true
		}
	}
	return false
}

// feed feeds the url into where it should be.
func (u *URL) feed(url *url.URL) {
	u.URL = url
}

// reset resets all fields in the u.
func (u *URL) reset() {
	u.URL = nil
	u.queryValues = nil
}

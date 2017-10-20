package air

import (
	"net/http"
	"time"
)

// Cookie represents the HTTP cookie.
type Cookie struct {
	Name     string
	Value    string
	Expires  time.Time
	MaxAge   int
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
}

// newCookie returns a new instance of the `Cookie`.
func newCookie(c *http.Cookie) *Cookie {
	return &Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Expires:  c.Expires,
		MaxAge:   c.MaxAge,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HTTPOnly: c.HttpOnly,
	}
}

// String returns the serialization string of the c.
func (c *Cookie) String() string {
	return (&http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Expires:  c.Expires,
		MaxAge:   c.MaxAge,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
	}).String()
}

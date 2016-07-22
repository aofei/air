package air

import (
	"time"

	"github.com/valyala/fasthttp"
)

type (
	// Cookie defines the interface for HTTP cookie.
	Cookie interface {
		// Name returns the name of the cookie.
		Name() string

		// SetName sets the name of the cookie.
		SetName(string)

		// Value returns the value of the cookie.
		Value() string

		// SetValue sets the value of the cookie.
		SetValue(string)

		// Path returns the path of the cookie.
		Path() string

		// SetPath sets the path of the cookie.
		SetPath(string)

		// Domain returns the domain of the cookie.
		Domain() string

		// SetDomain sets the domain of the cookie.
		SetDomain(string)

		// Expires returns the expiry time of the cookie.
		Expires() time.Time

		// SetExpires sets the expiry time of the cookie.
		SetExpires(time.Time)

		// Secure indicates if cookie is secured.
		Secure() bool

		// SetSecure sets the cookie as Secure.
		SetSecure(bool)

		// HTTPOnly indicate if cookies is HTTP only.
		HTTPOnly() bool

		// SetHTTPOnly sets the cookie as HTTPOnly.
		SetHTTPOnly(bool)
	}

	// FastCookie implements `Cookie`.
	FastCookie struct {
		*fasthttp.Cookie
	}
)

// Name implements `Cookie#Name` function.
func (c *FastCookie) Name() string {
	return string(c.Cookie.Key())
}

// SetName implements `Cookie#SetName` function.
func (c *FastCookie) SetName(name string) {
	c.Cookie.SetKey(name)
}

// Value implements `Cookie#Value` function.
func (c *FastCookie) Value() string {
	return string(c.Cookie.Value())
}

// SetValue implements `Cookie#SetValue` function.
func (c *FastCookie) SetValue(value string) {
	c.Cookie.SetValue(value)
}

// Path implements `Cookie#Path` function.
func (c *FastCookie) Path() string {
	return string(c.Cookie.Path())
}

// SetPath implements `Cookie#SetPath` function.
func (c *FastCookie) SetPath(path string) {
	c.Cookie.SetPath(path)
}

// Domain implements `Cookie#Domain` function.
func (c *FastCookie) Domain() string {
	return string(c.Cookie.Domain())
}

// SetDomain implements `Cookie#SetDomain` function.
func (c *FastCookie) SetDomain(domain string) {
	c.Cookie.SetDomain(domain)
}

// Expires implements `Cookie#Expires` function.
func (c *FastCookie) Expires() time.Time {
	return c.Cookie.Expire()
}

// SetExpires implements `Cookie#SetExpires` function.
func (c *FastCookie) SetExpires(expires time.Time) {
	c.Cookie.SetExpire(expires)
}

// Secure implements `Cookie#Secure` function.
func (c *FastCookie) Secure() bool {
	return c.Cookie.Secure()
}

// SetSecure implements `Cookie#SetSecure` function.
func (c *FastCookie) SetSecure(secure bool) {
	c.Cookie.SetSecure(secure)
}

// HTTPOnly implements `Cookie#HTTPOnly` function.
func (c *FastCookie) HTTPOnly() bool {
	return c.Cookie.HTTPOnly()
}

// SetHTTPOnly implements `Cookie#SetHTTPOnly` function.
func (c *FastCookie) SetHTTPOnly(httpOnly bool) {
	c.Cookie.SetHTTPOnly(httpOnly)
}

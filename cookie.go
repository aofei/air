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

	fastCookie struct {
		*fasthttp.Cookie
	}
)

// NewCookie creates an instance of `fastCookie`.
func NewCookie() *fastCookie {
	return &fastCookie{}
}

func (c *fastCookie) Name() string {
	return string(c.Cookie.Key())
}

func (c *fastCookie) SetName(name string) {
	c.Cookie.SetKey(name)
}

func (c *fastCookie) Value() string {
	return string(c.Cookie.Value())
}

func (c *fastCookie) SetValue(value string) {
	c.Cookie.SetValue(value)
}

func (c *fastCookie) Path() string {
	return string(c.Cookie.Path())
}

func (c *fastCookie) SetPath(path string) {
	c.Cookie.SetPath(path)
}

func (c *fastCookie) Domain() string {
	return string(c.Cookie.Domain())
}

func (c *fastCookie) SetDomain(domain string) {
	c.Cookie.SetDomain(domain)
}

func (c *fastCookie) Expires() time.Time {
	return c.Cookie.Expire()
}

func (c *fastCookie) SetExpires(expires time.Time) {
	c.Cookie.SetExpire(expires)
}

func (c *fastCookie) Secure() bool {
	return c.Cookie.Secure()
}

func (c *fastCookie) SetSecure(secure bool) {
	c.Cookie.SetSecure(secure)
}

func (c *fastCookie) HTTPOnly() bool {
	return c.Cookie.HTTPOnly()
}

func (c *fastCookie) SetHTTPOnly(httpOnly bool) {
	c.Cookie.SetHTTPOnly(httpOnly)
}

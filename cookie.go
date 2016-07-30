package air

import (
	"time"

	"github.com/valyala/fasthttp"
)

// Cookie for HTTP cookie.
type Cookie struct {
	fastCookie *fasthttp.Cookie
}

// Name returns the name of the cookie.
func (c *Cookie) Name() string {
	return string(c.fastCookie.Key())
}

// SetName sets the name of the cookie.
func (c *Cookie) SetName(name string) {
	c.fastCookie.SetKey(name)
}

// Value returns the value of the cookie.
func (c *Cookie) Value() string {
	return string(c.fastCookie.Value())
}

// SetValue sets the value of the cookie.
func (c *Cookie) SetValue(value string) {
	c.fastCookie.SetValue(value)
}

// Path returns the path of the cookie.
func (c *Cookie) Path() string {
	return string(c.fastCookie.Path())
}

// SetPath sets the path of the cookie.
func (c *Cookie) SetPath(path string) {
	c.fastCookie.SetPath(path)
}

// Domain returns the domain of the cookie.
func (c *Cookie) Domain() string {
	return string(c.fastCookie.Domain())
}

// SetDomain sets the domain of the cookie.
func (c *Cookie) SetDomain(domain string) {
	c.fastCookie.SetDomain(domain)
}

// Expires returns the expiry time of the cookie.
func (c *Cookie) Expires() time.Time {
	return c.fastCookie.Expire()
}

// SetExpires sets the expiry time of the cookie.
func (c *Cookie) SetExpires(expires time.Time) {
	c.fastCookie.SetExpire(expires)
}

// Secure indicates if cookie is secured.
func (c *Cookie) Secure() bool {
	return c.fastCookie.Secure()
}

// SetSecure sets the cookie as Secure.
func (c *Cookie) SetSecure(secure bool) {
	c.fastCookie.SetSecure(secure)
}

// HTTPOnly indicate if cookies is HTTP only.
func (c *Cookie) HTTPOnly() bool {
	return c.fastCookie.HTTPOnly()
}

// SetHTTPOnly sets the cookie as HTTPOnly.
func (c *Cookie) SetHTTPOnly(httpOnly bool) {
	c.fastCookie.SetHTTPOnly(httpOnly)
}

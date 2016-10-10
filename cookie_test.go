package air

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	a := New()
	c := a.Pool.Cookie()

	// Name
	c.SetName("name")
	assert.Equal(t, "name", c.Name())

	// Value
	c.SetValue("Aofei Sheng")
	assert.Equal(t, "Aofei Sheng", c.Value())

	// Path
	c.SetPath("/")
	assert.Equal(t, "/", c.Path())

	// Domain
	c.SetDomain("aofei.org")
	assert.Equal(t, "aofei.org", c.Domain())

	// Expires
	now := time.Now()
	c.SetExpires(now)
	assert.Equal(t, now, c.Expires())

	// Secure
	c.SetSecure(true)
	assert.Equal(t, true, c.Secure())

	// HTTPOnly
	c.SetHTTPOnly(true)
	assert.Equal(t, true, c.HTTPOnly())
}

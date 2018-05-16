package air

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	sc := &http.Cookie{
		Name:  "foo",
		Value: "bar",
	}

	c := &Cookie{
		Name:  sc.Name,
		Value: sc.Value,
	}

	assert.Equal(t, sc.Name, c.Name)
	assert.Equal(t, sc.Value, c.Value)
	assert.Equal(t, sc.String(), c.String())

	sc.Expires = time.Now().Add(time.Hour)
	sc.MaxAge = 3600
	sc.Domain = "example.com"
	sc.Path = "/"
	sc.Secure = true
	sc.HttpOnly = true

	c.Expires = sc.Expires
	c.MaxAge = sc.MaxAge
	c.Domain = sc.Domain
	c.Path = sc.Path
	c.Secure = sc.Secure
	c.HTTPOnly = sc.HttpOnly

	assert.Equal(t, sc.Expires, c.Expires)
	assert.Equal(t, sc.MaxAge, c.MaxAge)
	assert.Equal(t, sc.Domain, c.Domain)
	assert.Equal(t, sc.Path, c.Path)
	assert.Equal(t, sc.Secure, c.Secure)
	assert.Equal(t, sc.HttpOnly, c.HTTPOnly)
	assert.Equal(t, sc.String(), c.String())
}

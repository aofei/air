package air

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	sc := &http.Cookie{
		Name:     "foo",
		Value:    "bar",
		Expires:  time.Now().Add(time.Hour),
		MaxAge:   3600,
		Domain:   "example.com",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	}

	c := newCookie(sc)
	assert.Equal(t, sc.Name, c.Name)
	assert.Equal(t, sc.Value, c.Value)
	assert.Equal(t, sc.Expires, c.Expires)
	assert.Equal(t, sc.MaxAge, c.MaxAge)
	assert.Equal(t, sc.Domain, c.Domain)
	assert.Equal(t, sc.Path, c.Path)
	assert.Equal(t, sc.Secure, c.Secure)
	assert.Equal(t, sc.HttpOnly, c.HTTPOnly)
	assert.Equal(t, sc.String(), c.String())
}

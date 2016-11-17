package gases

import (
	"encoding/base64"

	"github.com/sheng/air"
)

type (
	// BasicAuthConfig defines the config for BasicAuth gas.
	BasicAuthConfig struct {
		// Skipper defines a function to skip gas.
		Skipper Skipper

		// Validator is a function to validate BasicAuth credentials.
		// Required.
		Validator BasicAuthValidator
	}

	// BasicAuthValidator defines a function to validate BasicAuth credentials.
	BasicAuthValidator func(string, string) bool
)

// DefaultBasicAuthConfig is the default BasicAuth gas config.
var DefaultBasicAuthConfig = BasicAuthConfig{
	Skipper: defaultSkipper,
}

// fill keeps all the fields of `BasicAuthConfig` have value.
func (c *BasicAuthConfig) fill() {
	if c.Skipper == nil {
		c.Skipper = DefaultBasicAuthConfig.Skipper
	}
	if c.Validator == nil {
		panic("basic-auth gas requires validator function")
	}
}

// BasicAuth returns an BasicAuth gas.
//
// For valid credentials it calls the next handler.
// For invalid credentials, it sends "401 - Unauthorized" response.
// For empty or invalid `Authorization` header, it sends "400 - Bad Request" response.
func BasicAuth(fn BasicAuthValidator) air.GasFunc {
	c := DefaultBasicAuthConfig
	c.Validator = fn
	return BasicAuthWithConfig(c)
}

const basic = "Basic"

// BasicAuthWithConfig returns an BasicAuth gas with config.
// See `BasicAuth()`.
func BasicAuthWithConfig(config BasicAuthConfig) air.GasFunc {
	// Defaults
	config.fill()

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			auth := c.Request.Header.Get(air.HeaderAuthorization)
			l := len(basic)

			if len(auth) > l+1 && auth[:l] == basic {
				b, err := base64.StdEncoding.DecodeString(auth[l+1:])
				if err != nil {
					return err
				}
				cred := string(b)
				for i := 0; i < len(cred); i++ {
					if cred[i] == ':' {
						// Verify credentials
						if config.Validator(cred[:i], cred[i+1:]) {
							return next(c)
						}
					}
				}
			}
			// Need to return `401` for browsers to pop-up login box.
			c.Header().Set(air.HeaderWWWAuthenticate, basic+" realm=Restricted")
			return air.ErrUnauthorized
		}
	}
}

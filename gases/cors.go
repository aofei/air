package gases

import (
	"strings"

	"github.com/sheng/air"
)

// CORSConfig defines the config for CORS gas.
type CORSConfig struct {
	// Skipper defines a function to skip gas.
	Skipper Skipper

	// AllowOrigin defines a list of origins that may access the resource.
	// Optional. If request header `Origin` is set, value is
	// []string{"<Origin>"} else []string{"*"}.
	AllowOrigins []string `json:"allow_origins"`

	// AllowHeaders defines a list of request headers that can be used when
	// making the actual request. This in response to a preflight request.
	// Optional. Default value []string{}.
	AllowHeaders []string `json:"allow_headers"`

	// AllowCredentials indicates whether or not the response to the request
	// can be exposed when the credentials flag is true. When used as part of
	// a response to a preflight request, this indicates whether or not the
	// actual request can be made using credentials.
	// Optional. Default value false.
	AllowCredentials bool `json:"allow_credentials"`

	// ExposeHeaders defines a whitelist headers that clients are allowed to
	// access.
	// Optional. Default value []string{}.
	ExposeHeaders []string `json:"expose_headers"`

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached.
	// Optional. Default value 0.
	MaxAge int `json:"max_age"`
}

// DefaultCORSConfig is the default CORS gas config.
var DefaultCORSConfig = CORSConfig{
	Skipper: defaultSkipper,
}

// fill keeps all the fields of `CORSConfig` have value.
func (c *CORSConfig) fill() {
	if c.Skipper == nil {
		c.Skipper = DefaultCORSConfig.Skipper
	}
}

// CORS returns a Cross-Origin Resource Sharing (CORS) gas.
// See: https://developer.mozilla.org/en/docs/Web/HTTP/Access_control_CORS
func CORS() air.GasFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig returns a CORS gas from config.
// See: `CORS()`.
func CORSWithConfig(config CORSConfig) air.GasFunc {
	// Defaults
	config.fill()

	allowedOrigins := strings.Join(config.AllowOrigins, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request
			origin := req.Header.Get(air.HeaderOrigin)

			if allowedOrigins == "" {
				if origin != "" {
					allowedOrigins = origin
				} else {
					if !config.AllowCredentials {
						allowedOrigins = "*"
					}
				}
			}

			c.Header().Add(air.HeaderVary, air.HeaderOrigin)
			c.Header().Set(air.HeaderAccessControlAllowOrigin, allowedOrigins)
			if config.AllowCredentials {
				c.Header().Set(air.HeaderAccessControlAllowCredentials, "true")
			}
			if exposeHeaders != "" {
				c.Header().Set(air.HeaderAccessControlExposeHeaders, exposeHeaders)
			}
			return next(c)
		}
	}
}

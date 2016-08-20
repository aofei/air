package gases

import (
	"strings"

	"github.com/sheng/air"
)

type (
	// CORSConfig defines the config for CORS gas.
	CORSConfig struct {
		// Skipper defines a function to skip gas.
		Skipper Skipper

		// AllowOrigin defines a list of origins that may access the resource.
		// Optional. Default value []string{"*"}.
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
)

var (
	// DefaultCORSConfig is the default CORS gas config.
	DefaultCORSConfig = CORSConfig{
		Skipper:      defaultSkipper,
		AllowOrigins: []string{"*"},
	}
)

// CORS returns a Cross-Origin Resource Sharing (CORS) gas.
// See: https://developer.mozilla.org/en/docs/Web/HTTP/Access_control_CORS
func CORS() air.GasFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig returns a CORS gas from config.
// See: `CORS()`.
func CORSWithConfig(config CORSConfig) air.GasFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultCORSConfig.Skipper
	}
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request
			res := c.Response
			origin := req.Header.Get(air.HeaderOrigin)
			originSet := req.Header.Contains(air.HeaderOrigin) // Issue #517

			// Check allowed origins
			allowedOrigin := ""
			for _, o := range config.AllowOrigins {
				if o == "*" || o == origin {
					allowedOrigin = o
					break
				}
			}

			res.Header.Add(air.HeaderVary, air.HeaderOrigin)
			if !originSet || allowedOrigin == "" {
				return next(c)
			}
			res.Header.Set(air.HeaderAccessControlAllowOrigin, allowedOrigin)
			if config.AllowCredentials {
				res.Header.Set(air.HeaderAccessControlAllowCredentials, "true")
			}
			if exposeHeaders != "" {
				res.Header.Set(air.HeaderAccessControlExposeHeaders, exposeHeaders)
			}
			return next(c)
		}
	}
}

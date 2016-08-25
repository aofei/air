package gases

import (
	"crypto/subtle"
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/sheng/air"
)

type (
	// CSRFConfig defines the config for CSRF gas.
	CSRFConfig struct {
		// Skipper defines a function to skip gas.
		Skipper Skipper

		// TokenLength is the length of the generated token.
		TokenLength uint8 `json:"token_length"`
		// Optional. Default value 32.

		// TokenLookup is a string in the form of "<source>:<key>" that is used
		// to extract token from the request.
		// Optional. Default value "header:X-CSRF-Token".
		// Possible values:
		// - "header:<name>"
		// - "form:<name>"
		// - "query:<name>"
		TokenLookup string `json:"token_lookup"`

		// Context key to store generated CSRF token into context.
		// Optional. Default value "csrf".
		ContextKey string `json:"context_key"`

		// Name of the CSRF cookie. This cookie will store CSRF token.
		// Optional. Default value "csrf".
		CookieName string `json:"cookie_name"`

		// Domain of the CSRF cookie.
		// Optional. Default value none.
		CookieDomain string `json:"cookie_domain"`

		// Path of the CSRF cookie.
		// Optional. Default value none.
		CookiePath string `json:"cookie_path"`

		// Max age (in seconds) of the CSRF cookie.
		// Optional. Default value 86400 (24hr).
		CookieMaxAge int `json:"cookie_max_age"`

		// Indicates if CSRF cookie is secure.
		// Optional. Default value false.
		CookieSecure bool `json:"cookie_secure"`

		// Indicates if CSRF cookie is HTTP only.
		// Optional. Default value false.
		CookieHTTPOnly bool `json:"cookie_http_only"`
	}

	// csrfTokenExtractor defines a function that takes `air.Context` and returns
	// either a token or an error.
	csrfTokenExtractor func(*air.Context) (string, error)
)

// DefaultCSRFConfig is the default CSRF gas config.
var DefaultCSRFConfig = CSRFConfig{
	Skipper:      defaultSkipper,
	TokenLength:  32,
	TokenLookup:  "header:" + air.HeaderXCSRFToken,
	ContextKey:   "csrf",
	CookieName:   "_csrf",
	CookieMaxAge: 86400,
}

// fill keeps all the fields of `CSRFConfig` have value.
func (c *CSRFConfig) fill() {
	if c.Skipper == nil {
		c.Skipper = DefaultCSRFConfig.Skipper
	}
	if c.TokenLength == 0 {
		c.TokenLength = DefaultCSRFConfig.TokenLength
	}
	if c.TokenLookup == "" {
		c.TokenLookup = DefaultCSRFConfig.TokenLookup
	}
	if c.ContextKey == "" {
		c.ContextKey = DefaultCSRFConfig.ContextKey
	}
	if c.CookieName == "" {
		c.CookieName = DefaultCSRFConfig.CookieName
	}
	if c.CookieMaxAge == 0 {
		c.CookieMaxAge = DefaultCSRFConfig.CookieMaxAge
	}
}

// CSRF returns a Cross-Site Request Forgery (CSRF) gas.
// See: https://en.wikipedia.org/wiki/Cross-site_request_forgery
func CSRF() air.GasFunc {
	c := DefaultCSRFConfig
	return CSRFWithConfig(c)
}

// CSRFWithConfig returns a CSRF gas from config.
// See `CSRF()`.
func CSRFWithConfig(config CSRFConfig) air.GasFunc {
	// Defaults
	config.fill()

	// Initialize
	parts := strings.Split(config.TokenLookup, ":")
	extractor := csrfTokenFromHeader(parts[1])
	switch parts[0] {
	case "form":
		extractor = csrfTokenFromForm(parts[1])
	case "query":
		extractor = csrfTokenFromQuery(parts[1])
	}

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request
			k, err := c.Cookie(config.CookieName)
			token := ""

			if err != nil {
				// Generate token
				token = randomString(config.TokenLength)
			} else {
				// Reuse token
				token = k.Value()
			}

			// Validate token only for requests which are not defined as 'safe' by RFC7231
			if req.Method() != air.GET {
				clientToken, err := extractor(c)
				if err != nil {
					return err
				}
				if !validateCSRFToken(token, clientToken) {
					return air.NewHTTPError(http.StatusForbidden, "csrf ioken is invalid")
				}
			}

			// Set CSRF cookie
			cookie := air.Cookie{}
			cookie.SetName(config.CookieName)
			cookie.SetValue(token)
			if config.CookiePath != "" {
				cookie.SetPath(config.CookiePath)
			}
			if config.CookieDomain != "" {
				cookie.SetDomain(config.CookieDomain)
			}
			cookie.SetExpires(time.Now().Add(time.Duration(config.CookieMaxAge) * time.Second))
			cookie.SetSecure(config.CookieSecure)
			cookie.SetHTTPOnly(config.CookieHTTPOnly)
			c.SetCookie(cookie)

			// Store token in the context
			c.SetValue(config.ContextKey, token)

			// Protect clients from caching the response
			c.Response.Header.Add(air.HeaderVary, air.HeaderCookie)

			return next(c)
		}
	}
}

// csrfTokenFromForm returns a `csrfTokenExtractor` that extracts token from the
// provided request header.
func csrfTokenFromHeader(header string) csrfTokenExtractor {
	return func(c *air.Context) (string, error) {
		return c.Request.Header.Get(header), nil
	}
}

// csrfTokenFromForm returns a `csrfTokenExtractor` that extracts token from the
// provided form parameter.
func csrfTokenFromForm(param string) csrfTokenExtractor {
	return func(c *air.Context) (string, error) {
		token := c.FormValue(param)
		if token == "" {
			return "", errors.New("empty csrf token in form param")
		}
		return token, nil
	}
}

// csrfTokenFromQuery returns a `csrfTokenExtractor` that extracts token from the
// provided query parameter.
func csrfTokenFromQuery(param string) csrfTokenExtractor {
	return func(c *air.Context) (string, error) {
		token := c.QueryParam(param)
		if token == "" {
			return "", errors.New("empty csrf token in query param")
		}
		return token, nil
	}
}

func validateCSRFToken(token, clientToken string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(clientToken)) == 1
}

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomString(length uint8) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = alphanumeric[rand.Int63()%int64(62)]
	}
	return string(b)
}

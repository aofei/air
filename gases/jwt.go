package gases

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sheng/air"

	"github.com/dgrijalva/jwt-go"
)

type (
	// JWTConfig defines the config for JWT gas.
	JWTConfig struct {
		// Skipper defines a function to skip gas.
		Skipper Skipper

		// Signing key to validate token.
		// Required.
		SigningKey interface{} `json:"signing_key"`

		// Signing method, used to check token signing method.
		// Optional. Default value HS256.
		SigningMethod string `json:"signing_method"`

		// Context key to store user information from the token into context.
		// Optional. Default value "user".
		ContextKey string `json:"context_key"`

		// Claims are extendable claims data defining token content.
		// Optional. Default value jwt.MapClaims
		Claims jwt.Claims

		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		TokenLookup string `json:"token_lookup"`
	}

	jwtExtractor func(*air.Context) (string, error)
)

const (
	bearer = "Bearer"

	// AlgorithmHS256 is the algorithm that checks token signing method.
	AlgorithmHS256 = "HS256"
)

// DefaultJWTConfig is the default JWT auth gas config.
var DefaultJWTConfig = JWTConfig{
	Skipper:       defaultSkipper,
	SigningMethod: AlgorithmHS256,
	ContextKey:    "user",
	Claims:        jwt.MapClaims{},
	TokenLookup:   "header:" + air.HeaderAuthorization,
}

// fill keeps all the fields of `JWTConfig` have value.
func (c *JWTConfig) fill() {
	if c.Skipper == nil {
		c.Skipper = DefaultJWTConfig.Skipper
	}
	if c.SigningKey == nil {
		panic("jwt gas requires signing key")
	}
	if c.SigningMethod == "" {
		c.SigningMethod = DefaultJWTConfig.SigningMethod
	}
	if c.ContextKey == "" {
		c.ContextKey = DefaultJWTConfig.ContextKey
	}
	if c.Claims == nil {
		c.Claims = DefaultJWTConfig.Claims
	}
	if c.TokenLookup == "" {
		c.TokenLookup = DefaultJWTConfig.TokenLookup
	}
}

// JWT returns a JSON Web Token (JWT) auth gas.
//
// For valid token, it sets the user in context and calls next handler.
// For invalid token, it returns "401 - Unauthorized" error.
// For empty token, it returns "400 - Bad Request" error.
//
// See: https://jwt.io/introduction
// See `JWTConfig.TokenLookup`
func JWT(key []byte) air.GasFunc {
	c := DefaultJWTConfig
	c.SigningKey = key
	return JWTWithConfig(c)
}

// JWTWithConfig returns a JWT auth gas from config.
// See: `JWT()`.
func JWTWithConfig(config JWTConfig) air.GasFunc {
	// Defaults
	config.fill()

	// Initialize
	parts := strings.Split(config.TokenLookup, ":")
	extractor := jwtFromHeader(parts[1])
	switch parts[0] {
	case "query":
		extractor = jwtFromQuery(parts[1])
	case "cookie":
		extractor = jwtFromCookie(parts[1])
	}

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			auth, err := extractor(c)
			if err != nil {
				return air.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			token, err := jwt.ParseWithClaims(auth, config.Claims, func(t *jwt.Token) (interface{}, error) {
				// Check the signing method
				if t.Method.Alg() != config.SigningMethod {
					return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
				}
				return config.SigningKey, nil

			})
			if err == nil && token.Valid {
				// Store user information from token into context.
				c.SetValue(config.ContextKey, token)
				return next(c)
			}
			return air.ErrUnauthorized
		}
	}
}

// jwtFromHeader returns a `jwtExtractor` that extracts token from request header.
func jwtFromHeader(header string) jwtExtractor {
	return func(c *air.Context) (string, error) {
		auth := c.Request.Header.Get(header)
		l := len(bearer)
		if len(auth) > l+1 && auth[:l] == bearer {
			return auth[l+1:], nil
		}
		return "", errors.New("empty or invalid jwt in request header")
	}
}

// jwtFromQuery returns a `jwtExtractor` that extracts token from query string.
func jwtFromQuery(param string) jwtExtractor {
	return func(c *air.Context) (string, error) {
		token := c.QueryParam(param)
		if token == "" {
			return "", errors.New("empty jwt in query string")
		}
		return token, nil
	}
}

// jwtFromCookie returns a `jwtExtractor` that extracts token from named cookie.
func jwtFromCookie(name string) jwtExtractor {
	return func(c *air.Context) (string, error) {
		cookie, err := c.Cookie(name)
		if err != nil {
			return "", errors.New("empty jwt in cookie")
		}
		return cookie.Value(), nil
	}
}

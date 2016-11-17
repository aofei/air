package gases

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/sheng/air"
)

type (
	// GzipConfig defines the config for Gzip gas.
	GzipConfig struct {
		// Skipper defines a function to skip gas.
		Skipper Skipper

		// Gzip compression level.
		// Optional. Default value -1.
		Level int `json:"level"`
	}

	gzipResponseWriter struct {
		http.ResponseWriter
	}
)

// DefaultGzipConfig is the default Gzip gas config.
var DefaultGzipConfig = GzipConfig{
	Skipper: defaultSkipper,
	Level:   -1,
}

// fill keeps all the fields of `GzipConfig` have value.
func (c *GzipConfig) fill() {
	if c.Skipper == nil {
		c.Skipper = DefaultGzipConfig.Skipper
	}
	if c.Level == 0 {
		c.Level = DefaultGzipConfig.Level
	}
}

// Gzip returns a gas which compresses HTTP response using Gzip compression
// scheme.
func Gzip() air.GasFunc {
	return GzipWithConfig(DefaultGzipConfig)
}

// GzipWithConfig return Gzip gas from config.
// See: `Gzip()`.
func GzipWithConfig(config GzipConfig) air.GasFunc {
	// Defaults
	config.fill()

	pool := gzipPool(config)
	scheme := "gzip"

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			c.Header().Add(air.HeaderVary, air.HeaderAcceptEncoding)
			if strings.Contains(c.Request.Header.Get(air.HeaderAcceptEncoding), scheme) {
				rw := c
				gw := pool.Get().(*gzip.Writer)
				gw.Reset(rw)
				defer func() {
					if c.Size == 0 {
						// We have to reset response to it's pristine state when
						// nothing is written to body or error is returned.
						// See issue #424, #407.
						c.SetWriter(rw)
						c.Header().Del(air.HeaderContentEncoding)
						gw.Reset(ioutil.Discard)
					}
					gw.Close()
					pool.Put(gw)
				}()
				g := gzipResponseWriter{ResponseWriter: c}
				c.Header().Set(air.HeaderContentEncoding, scheme)
				c.SetWriter(g)
			}
			return next(c)
		}
	}
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	if g.Header().Get(air.HeaderContentType) == "" {
		g.Header().Set(air.HeaderContentType, http.DetectContentType(b))
	}
	return g.Write(b)
}

func gzipPool(config GzipConfig) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(ioutil.Discard, config.Level)
			return w
		},
	}
}

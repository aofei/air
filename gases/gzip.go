package gases

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"air"
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
		air.Response
		io.Writer
	}
)

var (
	// DefaultGzipConfig is the default Gzip gas config.
	DefaultGzipConfig = GzipConfig{
		Skipper: defaultSkipper,
		Level:   -1,
	}
)

// Gzip returns a gas which compresses HTTP response using Gzip compression
// scheme.
func Gzip() air.GasFunc {
	return GzipWithConfig(DefaultGzipConfig)
}

// GzipWithConfig return Gzip gas from config.
// See: `Gzip()`.
func GzipWithConfig(config GzipConfig) air.GasFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultGzipConfig.Skipper
	}
	if config.Level == 0 {
		config.Level = DefaultGzipConfig.Level
	}

	pool := gzipPool(config)
	scheme := "gzip"

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			res := c.Response()
			res.Header().Add(air.HeaderVary, air.HeaderAcceptEncoding)
			if strings.Contains(c.Request().Header().Get(air.HeaderAcceptEncoding), scheme) {
				rw := res.Writer()
				gw := pool.Get().(*gzip.Writer)
				gw.Reset(rw)
				defer func() {
					if res.Size() == 0 {
						// We have to reset response to it's pristine state when
						// nothing is written to body or error is returned.
						// See issue #424, #407.
						res.SetWriter(rw)
						res.Header().Del(air.HeaderContentEncoding)
						gw.Reset(ioutil.Discard)
					}
					gw.Close()
					pool.Put(gw)
				}()
				g := gzipResponseWriter{Response: res, Writer: gw}
				res.Header().Set(air.HeaderContentEncoding, scheme)
				res.SetWriter(g)
			}
			return next(c)
		}
	}
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	if g.Header().Get(air.HeaderContentType) == "" {
		g.Header().Set(air.HeaderContentType, http.DetectContentType(b))
	}
	return g.Writer.Write(b)
}

func gzipPool(config GzipConfig) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(ioutil.Discard, config.Level)
			return w
		},
	}
}

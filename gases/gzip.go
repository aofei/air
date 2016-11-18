package gases

import (
	"bufio"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

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
		io.Writer
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

	scheme := "gzip"

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			c.Header().Add(air.HeaderVary, air.HeaderAcceptEncoding)
			if strings.Contains(c.Request.Header.Get(air.HeaderAcceptEncoding), scheme) {
				rw := c.ResponseWriter
				w, err := gzip.NewWriterLevel(rw, config.Level)
				if err != nil {
					return err
				}
				defer func() {
					if c.Size == 0 {
						c.ResponseWriter = rw
						c.Header().Del(air.HeaderContentEncoding)
						w.Reset(ioutil.Discard)
					}
					w.Close()
				}()
				grw := &gzipResponseWriter{Writer: w, ResponseWriter: rw}
				c.Header().Set(air.HeaderContentEncoding, scheme)
				c.ResponseWriter = grw
			}
			return next(c)
		}
	}
}

func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	if grw.Header().Get(air.HeaderContentType) == "" {
		grw.Header().Set(air.HeaderContentType, http.DetectContentType(b))
	}
	return grw.Writer.Write(b)
}

func (grw *gzipResponseWriter) Flush() error {
	return grw.Writer.(*gzip.Writer).Flush()
}

func (grw *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return grw.ResponseWriter.(http.Hijacker).Hijack()
}

func (grw *gzipResponseWriter) CloseNotify() <-chan bool {
	return grw.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

package gases

import (
	"bytes"
	"io"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/sheng/air"
)

// LoggerConfig defines the config for Logger gas.
type LoggerConfig struct {
	template   *template.Template
	bufferPool *sync.Pool

	// Skipper defines a function to skip gas.
	Skipper Skipper

	// Log format which can be constructed using the following tags:
	//
	// - time_rfc3339
	// - id (Request ID - Not implemented)
	// - remote_ip
	// - request_uri
	// - host
	// - method
	// - path
	// - referer
	// - user_agent
	// - status
	// - latency (In microseconds)
	// - latency_human (Human readable)
	// - bytes_in (Bytes received)
	// - bytes_out (Bytes sent)
	//
	// Example "{{.remote_ip}} {{.status}}"
	//
	// Optional. Default value DefaultLoggerConfig.Format.
	Format string `json:"format"`

	// Output is a writer where logs are written.
	// Optional. Default value os.Stdout.
	Output io.Writer
}

// DefaultLoggerConfig is the default Logger gas config.
var DefaultLoggerConfig = LoggerConfig{
	Skipper: defaultSkipper,
	Format: `{"time":"{{.time_rfc3339}}","remote_ip":"{{.remote_ip}}",` +
		`"method":"{{.method}}","path":"{{.path}}","status":{{.status}},` +
		`"latency":{{.latency}},"latency_human":"{{.latency_human}}",` +
		`"bytes_in":{{.bytes_in}},"bytes_out":{{.bytes_out}}}` + "\n",
	Output: os.Stdout,
}

// fill keeps all the fields of `LoggerConfig` have value.
func (c *LoggerConfig) fill() {
	if c.Skipper == nil {
		c.Skipper = DefaultLoggerConfig.Skipper
	}
	if c.Format == "" {
		c.Format = DefaultLoggerConfig.Format
	}
	if c.Output == nil {
		c.Output = DefaultLoggerConfig.Output
	}
}

// Logger returns a gas that logs HTTP requests.
func Logger() air.GasFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger gas from config.
// See: `Logger()`.
func LoggerWithConfig(config LoggerConfig) air.GasFunc {
	// Defaults
	config.fill()

	config.template, _ = template.New("logger").Parse(config.Format)
	config.bufferPool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 256))
		},
	}

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c *air.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request
			res := c.Response
			start := time.Now()
			if err = next(c); err != nil {
				c.Air.HTTPErrorHandler(err, c)
			}
			stop := time.Now()
			buf := config.bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			defer config.bufferPool.Put(buf)

			data := make(air.JSONMap)
			data["time_rfc3339"] = time.Now().Format(time.RFC3339)
			data["remote_ip"] = req.RemoteIP()
			data["host"] = req.Host()
			data["request_uri"] = req.RequestURI()
			data["method"] = req.Method()
			p := req.URI.Path()
			if p == "" {
				p = "/"
			}
			data["path"] = p
			data["referer"] = req.Referer()
			data["user_agent"] = req.UserAgent()
			data["status"] = c.StatusCode
			data["latency"] = stop.Sub(start).Nanoseconds() / 1000
			data["latency_human"] = stop.Sub(start).String()
			b := req.Header.Get(air.HeaderContentLength)
			if b == "" {
				b = "0"
			}
			data["bytes_in"] = b
			data["bytes_out"] = res.Size
			err = config.template.Execute(buf, data)
			if err == nil {
				config.Output.Write(buf.Bytes())
			}
			return
		}
	}
}

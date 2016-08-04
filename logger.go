package air

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/valyala/fasttemplate"
)

// Logger is used to log information generated in runtime.
type (
	Logger struct {
		template   *fasttemplate.Template
		bufferPool sync.Pool
		mutex      sync.Mutex
		levels     []string

		Prefix string
		Level  LoggerLevel
		Output io.Writer
	}

	// LoggerLevel is level of the logger.
	LoggerLevel uint8
)

// Logger levels
const (
	DEBUG LoggerLevel = iota
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

// NewLogger returns an new instance of `Logger`.
func NewLogger(prefix string) *Logger {
	l := &Logger{
		Level:  INFO,
		Prefix: prefix,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 256))
			},
		},
	}
	l.template = l.newTemplate(`{"time":"${time_rfc3339}","level":"${level}","prefix":"${prefix}",` +
		`"file":"${short_file}","line":"${line}"}`)
	l.initLevels()
	l.Output = os.Stdout
	return l
}

// initLevels initializes the logger levels.
func (l *Logger) initLevels() {
	l.levels = []string{
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"FATAL",
	}
}

// newTemplate returns an new instance of `fasttemplate.Template`.
func (l *Logger) newTemplate(format string) *fasttemplate.Template {
	return fasttemplate.New(format, "${", "}")
}

// SetFormat sets log format of the logger.
func (l *Logger) SetFormat(format string) {
	l.template = l.newTemplate(format)
}

// Print prints log info with provided type i.
func (l *Logger) Print(i ...interface{}) {
	fmt.Fprintln(l.Output, i...)
}

// Print prints log info in a format with provided type i.
func (l *Logger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.Output, f, args...)
}

// Printj prints log info with provided json map.
func (l *Logger) Printj(j map[string]interface{}) {
	json.NewEncoder(l.Output).Encode(j)
}

// Debug prints debug level log info with provided type i.
func (l *Logger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

// Debugf prints debug level log info in a format with provided type i.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Debugj prints debug level log info in a format with provided json map.
func (l *Logger) Debugj(j map[string]interface{}) {
	l.log(DEBUG, "json", j)
}

// Info prints info level log info with provided type i.
func (l *Logger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

// Infof prints info level log info in a format with provided type i.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Infoj prints info level log info in a format with provided json map.
func (l *Logger) Infoj(j map[string]interface{}) {
	l.log(INFO, "json", j)
}

// Warn prints warn level log info with provided type i.
func (l *Logger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

// Warnf prints warn level log info in a format with provided type i.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Warnj prints warn level log info in a format with provided json map.
func (l *Logger) Warnj(j map[string]interface{}) {
	l.log(WARN, "json", j)
}

// Error prints error level log info with provided type i.
func (l *Logger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

// Errorf prints error level log info in a format with provided type i.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Errorj prints error level log info in a format with provided json map.
func (l *Logger) Errorj(j map[string]interface{}) {
	l.log(ERROR, "json", j)
}

// Fatal prints fatal level log info with provided type i.
func (l *Logger) Fatal(i ...interface{}) {
	l.log(FATAL, "", i...)
	os.Exit(1)
}

// Fatalf prints fatal level log info in a format with provided type i.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

// Fatalj prints fatal level log info in a format with provided json map.
func (l *Logger) Fatalj(j map[string]interface{}) {
	l.log(FATAL, "json", j)
}

// log prints log info in a format with provided level and args.
func (l *Logger) log(lvl LoggerLevel, format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	buf := l.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer l.bufferPool.Put(buf)
	_, file, line, _ := runtime.Caller(3)

	if lvl >= l.Level {
		message := ""
		if format == "" {
			message = fmt.Sprint(args...)
		} else if format == "json" {
			b, err := json.Marshal(args[0])
			if err != nil {
				panic(err)
			}
			message = string(b)
		} else {
			message = fmt.Sprintf(format, args...)
		}

		if lvl >= ERROR {
			// panic(message)
		}

		_, err := l.template.ExecuteFunc(buf, func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "time_rfc3339":
				return w.Write([]byte(time.Now().Format(time.RFC3339)))
			case "level":
				return w.Write([]byte(l.levels[lvl]))
			case "prefix":
				return w.Write([]byte(l.Prefix))
			case "long_file":
				return w.Write([]byte(file))
			case "short_file":
				return w.Write([]byte(path.Base(file)))
			case "line":
				return w.Write([]byte(strconv.Itoa(line)))
			}
			return 0, nil
		})

		if err == nil {
			s := buf.String()
			i := buf.Len() - 1
			if s[i] == '}' {
				// JSON header
				buf.Truncate(i)
				buf.WriteByte(',')
				if format == "json" {
					buf.WriteString(message[1:])
				} else {
					buf.WriteString(`"message":"`)
					buf.WriteString(message)
					buf.WriteString(`"}`)
				}
			} else {
				// Text header
				buf.WriteByte(' ')
				buf.WriteString(message)
			}
			buf.WriteByte('\n')
			l.Output.Write(buf.Bytes())
		}
	}
}

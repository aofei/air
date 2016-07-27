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

type (
	// Logger defines the logging interface.
	Logger interface {
		SetOutput(io.Writer)
		SetLevel(LoggerLevel)
		Print(...interface{})
		Printf(string, ...interface{})
		Printj(JSON)
		Debug(...interface{})
		Debugf(string, ...interface{})
		Debugj(JSON)
		Info(...interface{})
		Infof(string, ...interface{})
		Infoj(JSON)
		Warn(...interface{})
		Warnf(string, ...interface{})
		Warnj(JSON)
		Error(...interface{})
		Errorf(string, ...interface{})
		Errorj(JSON)
		Fatal(...interface{})
		Fatalj(JSON)
		Fatalf(string, ...interface{})
	}

	airLogger struct {
		prefix     string
		level      LoggerLevel
		output     io.Writer
		template   *fasttemplate.Template
		levels     []string
		bufferPool sync.Pool
		mutex      sync.Mutex
	}

	// LoggerLevel is a level of the Logger
	LoggerLevel uint8

	// JSON for Logger output format
	JSON map[string]interface{}
)

// LoggerLevel
const (
	DEBUG LoggerLevel = iota
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

var defaultHeader = `{"time":"${time_rfc3339}","level":"${level}","prefix":"${prefix}",` +
	`"file":"${short_file}","line":"${line}"}`

// NewLogger creates an instance of `Logger`
func NewLogger(prefix string) Logger {
	l := &airLogger{
		level:  INFO,
		prefix: prefix,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 256))
			},
		},
	}
	l.template = l.newTemplate(defaultHeader)
	l.initLevels()
	l.SetOutput(os.Stdout)
	return l
}

func (l *airLogger) initLevels() {
	l.levels = []string{
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"FATAL",
	}
}

func (l *airLogger) newTemplate(format string) *fasttemplate.Template {
	return fasttemplate.New(format, "${", "}")
}

func (l *airLogger) Prefix() string {
	return l.prefix
}

func (l *airLogger) SetPrefix(p string) {
	l.prefix = p
}

func (l *airLogger) Level() LoggerLevel {
	return l.level
}

func (l *airLogger) SetLevel(v LoggerLevel) {
	l.level = v
}

func (l *airLogger) Output() io.Writer {
	return l.output
}

func (l *airLogger) SetHeader(h string) {
	l.template = l.newTemplate(h)
}

func (l *airLogger) SetOutput(w io.Writer) {
	l.output = w
}

func (l *airLogger) Print(i ...interface{}) {
	fmt.Fprintln(l.output, i...)
}

func (l *airLogger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.output, f, args...)
}

func (l *airLogger) Printj(j JSON) {
	json.NewEncoder(l.output).Encode(j)
}

func (l *airLogger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

func (l *airLogger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *airLogger) Debugj(j JSON) {
	l.log(DEBUG, "json", j)
}

func (l *airLogger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

func (l *airLogger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *airLogger) Infoj(j JSON) {
	l.log(INFO, "json", j)
}

func (l *airLogger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

func (l *airLogger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *airLogger) Warnj(j JSON) {
	l.log(WARN, "json", j)
}

func (l *airLogger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

func (l *airLogger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *airLogger) Errorj(j JSON) {
	l.log(ERROR, "json", j)
}

func (l *airLogger) Fatal(i ...interface{}) {
	l.log(FATAL, "", i...)
	os.Exit(1)
}

func (l *airLogger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

func (l *airLogger) Fatalj(j JSON) {
	l.log(FATAL, "json", j)
}

func (l *airLogger) log(v LoggerLevel, format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	buf := l.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer l.bufferPool.Put(buf)
	_, file, line, _ := runtime.Caller(3)

	if v >= l.level {
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

		if v >= ERROR {
			// panic(message)
		}

		_, err := l.template.ExecuteFunc(buf, func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "time_rfc3339":
				return w.Write([]byte(time.Now().Format(time.RFC3339)))
			case "level":
				return w.Write([]byte(l.levels[v]))
			case "prefix":
				return w.Write([]byte(l.prefix))
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
			l.output.Write(buf.Bytes())
		}
	}
}

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
	"text/template"
	"time"
)

type (
	// Logger is used to log information generated in the runtime.
	Logger struct {
		template   *template.Template
		bufferPool *sync.Pool
		mutex      *sync.Mutex
		levels     []string
		air        *Air

		Level  LoggerLevel
		Output io.Writer
	}

	// LoggerLevel is the level of the `Logger`.
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

// newLogger returns a pointer of a new instance of the `Logger`.
func newLogger(a *Air) *Logger {
	l := &Logger{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 256))
			},
		},
		mutex: &sync.Mutex{},
		levels: []string{
			"DEBUG",
			"INFO",
			"WARN",
			"ERROR",
			"FATAL",
		},
		air:   a,
		Level: OFF,
	}
	l.template, _ = template.New("logger").Parse(a.Config.LogFormat)
	l.Output = os.Stdout
	return l
}

// Print prints log info with the provided type i.
func (l *Logger) Print(i ...interface{}) {
	fmt.Fprintln(l.Output, i...)
}

// Printf prints log info in a format with the provided args.
func (l *Logger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.Output, f, args...)
}

// Printj prints log info with the provided JSON map i.
func (l *Logger) Printj(j JSONMap) {
	json.NewEncoder(l.Output).Encode(j)
}

// Debug prints debug level log info with the provided type i.
func (l *Logger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

// Debugf prints debug level log info in a format with the provided args.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Debugj prints debug level log info in a format with the provided JSON map i.
func (l *Logger) Debugj(j JSONMap) {
	l.log(DEBUG, "json", j)
}

// Info prints info level log info with the provided type i.
func (l *Logger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

// Infof prints info level log info in a format with the provided args.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Infoj prints info level log info in a format with the provided JSON map i.
func (l *Logger) Infoj(j JSONMap) {
	l.log(INFO, "json", j)
}

// Warn prints warn level log info with the provided type i.
func (l *Logger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

// Warnf prints warn level log info in a format with the provided args.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Warnj prints warn level log info in a format with the provided JSON map i.
func (l *Logger) Warnj(j JSONMap) {
	l.log(WARN, "json", j)
}

// Error prints error level log info with the provided type i.
func (l *Logger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

// Errorf prints error level log info in a format with the provided args.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Errorj prints error level log info in a format with the provided JSON map i.
func (l *Logger) Errorj(j JSONMap) {
	l.log(ERROR, "json", j)
}

// Fatal prints fatal level log info with the provided type i.
func (l *Logger) Fatal(i ...interface{}) {
	l.log(FATAL, "", i...)
	os.Exit(1)
}

// Fatalf prints fatal level log info in a format with the provided args.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

// Fatalj prints fatal level log info in a format with the provided JSON map i.
func (l *Logger) Fatalj(j JSONMap) {
	l.log(FATAL, "json", j)
}

// log prints log info in a format with the provided level lvl and the args.
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

		if lvl == FATAL {
			panic(message)
		}

		data := make(JSONMap)
		data["app_name"] = l.air.Config.AppName
		data["time_rfc3339"] = time.Now().Format(time.RFC3339)
		data["level"] = l.levels[lvl]
		data["long_file"] = file
		data["short_file"] = path.Base(file)
		data["line"] = strconv.Itoa(line)
		err := l.template.Execute(buf, data)

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

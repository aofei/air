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

// Logger is used to log information generated in the runtime.
type Logger struct {
	air *Air

	template   *template.Template
	bufferPool *sync.Pool
	mutex      *sync.Mutex
	levels     []string

	Output io.Writer
}

// loggerLevel is the level of the `Logger`.
type loggerLevel uint8

// logger levels
const (
	lvlDebug loggerLevel = iota
	lvlInfo
	lvlWarn
	lvlError
	lvlFatal
)

// newLogger returns a pointer of a new instance of the `Logger`.
func newLogger(a *Air) *Logger {
	return &Logger{
		air: a,
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
		Output: os.Stdout,
	}
}

// Print prints the log info with the provided type i.
func (l *Logger) Print(i ...interface{}) {
	fmt.Fprintln(l.Output, i...)
}

// Printf prints the log info in the format with the provided args.
func (l *Logger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.Output, f, args...)
}

// Printj prints the log info in the JSON format with the provided m.
func (l *Logger) Printj(m map[string]interface{}) {
	json.NewEncoder(l.Output).Encode(m)
}

// Debug prints the DEBUG level log info with the provided type i.
func (l *Logger) Debug(i ...interface{}) {
	l.log(lvlDebug, "", i...)
}

// Debugf prints the DEBUG level log info in the format with the provided args.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(lvlDebug, format, args...)
}

// Debugj prints the DEBUG level log info in the JSON format with the provided m.
func (l *Logger) Debugj(m map[string]interface{}) {
	l.log(lvlDebug, "json", m)
}

// Info prints the INFO level log info with the provided type i.
func (l *Logger) Info(i ...interface{}) {
	l.log(lvlInfo, "", i...)
}

// Infof prints the INFO level log info in the format with the provided args.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(lvlInfo, format, args...)
}

// Infoj prints the INFO level log info in the JSON format with the provided m.
func (l *Logger) Infoj(m map[string]interface{}) {
	l.log(lvlInfo, "json", m)
}

// Warn prints the WARN level log info with the provided type i.
func (l *Logger) Warn(i ...interface{}) {
	l.log(lvlWarn, "", i...)
}

// Warnf prints the WARN level log info in the format with the provided args.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(lvlWarn, format, args...)
}

// Warnj prints the WARN level log info in the JSON format with the provided m.
func (l *Logger) Warnj(m map[string]interface{}) {
	l.log(lvlWarn, "json", m)
}

// Error prints the ERROR level log info with the provided type i.
func (l *Logger) Error(i ...interface{}) {
	l.log(lvlError, "", i...)
}

// Errorf prints the ERROR level log info in the format with the provided args.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(lvlError, format, args...)
}

// Errorj prints the ERROR level log info in the JSON format with the provided m.
func (l *Logger) Errorj(m map[string]interface{}) {
	l.log(lvlError, "json", m)
}

// Fatal prints the FATAL level log info with the provided type i.
func (l *Logger) Fatal(i ...interface{}) {
	l.log(lvlFatal, "", i...)
	os.Exit(1)
}

// Fatalf prints the FATAL level log info in the format with the provided args.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(lvlFatal, format, args...)
	os.Exit(1)
}

// Fatalj prints the FATAL level log info in the JSON format with the provided m.
func (l *Logger) Fatalj(m map[string]interface{}) {
	l.log(lvlFatal, "json", m)
	os.Exit(1)
}

// log prints the lvl level log info in the format with the args.
func (l *Logger) log(lvl loggerLevel, format string, args ...interface{}) {
	if !l.air.LoggerEnabled {
		return
	} else if l.template == nil {
		l.template = template.Must(
			template.New("logger").Parse(l.air.LoggerFormat),
		)
	}

	l.mutex.Lock()
	buf := l.bufferPool.Get().(*bytes.Buffer)

	message := ""
	if format == "" {
		message = fmt.Sprint(args...)
	} else if format == "json" {
		b, _ := json.Marshal(args[0])
		message = string(b)
	} else {
		message = fmt.Sprintf(format, args...)
	}

	if lvl == lvlFatal {
		panic(message)
	}

	_, file, line, _ := runtime.Caller(3)

	data := map[string]interface{}{}
	data["app_name"] = l.air.AppName
	data["time_rfc3339"] = time.Now().Format(time.RFC3339)
	data["level"] = l.levels[lvl]
	data["short_file"] = path.Base(file)
	data["long_file"] = file
	data["line"] = strconv.Itoa(line)

	if err := l.template.Execute(buf, data); err == nil {
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

	buf.Reset()
	l.bufferPool.Put(buf)
	l.mutex.Unlock()
}

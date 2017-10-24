package air

import (
	"bytes"
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
	Output io.Writer

	air      *Air
	mutex    *sync.Mutex
	template *template.Template
}

// newLogger returns a new instance of the `Logger`.
func newLogger(a *Air) *Logger {
	return &Logger{
		Output: os.Stdout,
		air:    a,
		mutex:  &sync.Mutex{},
		template: template.Must(
			template.New("logger").Parse(a.LoggerFormat),
		),
	}
}

// Debug logs the v at the DEBUG level.
func (l *Logger) Debug(v ...interface{}) {
	l.log("DEBUG", v...)
}

// Info logs the v at the INFO level.
func (l *Logger) Info(v ...interface{}) {
	l.log("INFO", v...)
}

// Warn logs the v at the WARN level.
func (l *Logger) Warn(v ...interface{}) {
	l.log("WARN", v...)
}

// Error logs the v at the ERROR level.
func (l *Logger) Error(v ...interface{}) {
	l.log("ERROR", v...)
}

// Panic logs the v at the PANIC level.
func (l *Logger) Panic(v ...interface{}) {
	l.log("PANIC", v...)
	panic(fmt.Sprint(v...))
}

// Fatal logs the v at the FATAL level.
func (l *Logger) Fatal(v ...interface{}) {
	l.log("FATAL", v...)
	os.Exit(1)
}

// log logs the v at the level.
func (l *Logger) log(level string, v ...interface{}) {
	if !l.air.LoggerEnabled {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	_, file, line, _ := runtime.Caller(3)

	values := map[string]interface{}{}
	values["app_name"] = l.air.AppName
	values["time_rfc3339"] = time.Now().UTC().Format(time.RFC3339)
	values["level"] = level
	values["short_file"] = path.Base(file)
	values["long_file"] = file
	values["line"] = strconv.Itoa(line)

	buf := &bytes.Buffer{}
	if err := l.template.Execute(buf, values); err == nil {
		message := fmt.Sprint(v...)
		if i := buf.Len() - 1; buf.String()[i] == '}' { // JSON
			buf.Truncate(i)
			buf.WriteByte(',')
			buf.WriteString(`"message":"`)
			buf.WriteString(message)
			buf.WriteString(`"}`)
		} else { // Text
			buf.WriteByte(' ')
			buf.WriteString(message)
		}
		buf.WriteByte('\n')
		l.Output.Write(buf.Bytes())
	}
}

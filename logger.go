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
	Logger interface {
		// Output returns the output of the `Logger`.
		Output() io.Writer

		// SetOutput sets the w to the output of the `Logger`.
		SetOutput(w io.Writer)

		// Print prints the log info with the provided type i.
		Print(i ...interface{})

		// Printf prints the log info in the format with the provided args.
		Printf(format string, args ...interface{})

		// Printj prints the log info in the JSON format with the provided m.
		Printj(m Map)

		// Debug prints the DEBUG level log info with the provided type i.
		Debug(i ...interface{})

		// Debugf prints the DEBUG level log info in the format with the provided args.
		Debugf(format string, args ...interface{})

		// Debugj prints the DEBUG level log info in the JSON format with the provided m.
		Debugj(m Map)

		// Info prints the INFO level log info with the provided type i.
		Info(i ...interface{})

		// Infof prints the INFO level log info in the format with the provided args.
		Infof(format string, args ...interface{})

		// Infoj prints the INFO level log info in the JSON format with the provided m.
		Infoj(m Map)

		// Warn prints the WARN level log info with the provided type i.
		Warn(i ...interface{})

		// Warnf prints the WARN level log info in the format with the provided args.
		Warnf(format string, args ...interface{})

		// Warnj prints the WARN level log info in the JSON format with the provided m.
		Warnj(m Map)

		// Error prints the ERROR level log info with the provided type i.
		Error(i ...interface{})

		// Errorf prints the ERROR level log info in the format with the provided args.
		Errorf(format string, args ...interface{})

		// Errorj prints the ERROR level log info in the JSON format with the provided m.
		Errorj(m Map)

		// Fatal prints the FATAL level log info with the provided type i.
		Fatal(i ...interface{})

		// Fatalf prints the FATAL level log info in the format with the provided args.
		Fatalf(format string, args ...interface{})

		// Fatalj prints the FATAL level log info in the JSON format with the provided m.
		Fatalj(m Map)
	}

	// logger implements the `Logger` by using the `template.Template`.
	logger struct {
		air *Air

		template   *template.Template
		bufferPool *sync.Pool
		mutex      *sync.Mutex

		enabled bool
		levels  []string
		output  io.Writer
	}

	// loggerLevel is the level of the `logger`.
	loggerLevel uint8
)

// logger levels
const (
	lvlDebug loggerLevel = iota
	lvlInfo
	lvlWarn
	lvlError
	lvlFatal
)

// newLogger returns a pointer of a new instance of the `logger`.
func newLogger(a *Air) *logger {
	l := &logger{air: a}

	l.bufferPool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 256))
		},
	}
	l.mutex = &sync.Mutex{}

	l.levels = []string{
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"FATAL",
	}
	l.output = os.Stdout

	return l
}

// Output implements the `Logger#Output()`.
func (l *logger) Output() io.Writer {
	return l.output
}

// SetOutput implements the `Logger#SetOutput()`.
func (l *logger) SetOutput(w io.Writer) {
	l.output = w
}

// Print implements the `Logger#Print()` by using the `template.Template`.
func (l *logger) Print(i ...interface{}) {
	fmt.Fprintln(l.output, i...)
}

// Printf implements the `Logger#Printf()` by using the `template.Template`.
func (l *logger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.output, f, args...)
}

// Printj implements the `Logger#Printj()` by using the `template.Template`.
func (l *logger) Printj(m Map) {
	json.NewEncoder(l.output).Encode(m)
}

// Debug implements the `Logger#Debug()` by using the `template.Template`.
func (l *logger) Debug(i ...interface{}) {
	l.log(lvlDebug, "", i...)
}

// Debugf implements the `Logger#Debugf()` by using the `template.Template`.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.log(lvlDebug, format, args...)
}

// Debugj implements the `Logger#Debugj()` by using the `template.Template`.
func (l *logger) Debugj(m Map) {
	l.log(lvlDebug, "json", m)
}

// Info implements the `Logger#Info()` by using the `template.Template`.
func (l *logger) Info(i ...interface{}) {
	l.log(lvlInfo, "", i...)
}

// Infof implements the `Logger#Infof()` by using the `template.Template`.
func (l *logger) Infof(format string, args ...interface{}) {
	l.log(lvlInfo, format, args...)
}

// Infoj implements the `Logger#Infoj()` by using the `template.Template`.
func (l *logger) Infoj(m Map) {
	l.log(lvlInfo, "json", m)
}

// Warn implements the `Logger#Warn()` by using the `template.Template`.
func (l *logger) Warn(i ...interface{}) {
	l.log(lvlWarn, "", i...)
}

// Warnf implements the `Logger#Warnf()` by using the `template.Template`.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.log(lvlWarn, format, args...)
}

// Warnj implements the `Logger#Warnj()` by using the `template.Template`.
func (l *logger) Warnj(m Map) {
	l.log(lvlWarn, "json", m)
}

// Error implements the `Logger#Error()` by using the `template.Template`.
func (l *logger) Error(i ...interface{}) {
	l.log(lvlError, "", i...)
}

// Errorf implements the `Logger#Errorf()` by using the `template.Template`.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.log(lvlError, format, args...)
}

// Errorj implements the `Logger#Errorj()` by using the `template.Template`.
func (l *logger) Errorj(m Map) {
	l.log(lvlError, "json", m)
}

// Fatal implements the `Logger#Fatal()` by using the `template.Template`.
func (l *logger) Fatal(i ...interface{}) {
	l.log(lvlFatal, "", i...)
	os.Exit(1)
}

// Fatalf implements the `Logger#Fatalf()` by using the `template.Template`.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.log(lvlFatal, format, args...)
	os.Exit(1)
}

// Fatalj implements the `Logger#Fatalj()` by using the `template.Template`.
func (l *logger) Fatalj(m Map) {
	l.log(lvlFatal, "json", m)
	os.Exit(1)
}

// log prints the lvl level log info in the format with the args.
func (l *logger) log(lvl loggerLevel, format string, args ...interface{}) {
	if !l.air.Config.LoggerEnabled {
		return
	} else if l.template == nil {
		l.template = template.Must(template.New("logger").Parse(l.air.Config.LogFormat))
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

	data := Map{}
	data["app_name"] = l.air.Config.AppName
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
		l.output.Write(buf.Bytes())
	}

	buf.Reset()
	l.bufferPool.Put(buf)
	l.mutex.Unlock()
}

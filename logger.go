package air

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// logger is an active logging object that generates lines of output.
type logger struct {
	sync.Mutex

	a *Air
}

// newLogger returns a new instance of the `logger` with the a.
func newLogger(a *Air) *logger {
	return &logger{
		a: a,
	}
}

// log logs the m at the ll with the optional es.
func (l *logger) log(ll LoggerLevel, m string, es ...map[string]interface{}) {
	if !l.a.DebugMode && ll < l.a.LoggerLevel {
		return
	}

	l.Lock()
	defer l.Unlock()

	fs := map[string]interface{}{
		"app_name": l.a.AppName,
		"time":     time.Now().UnixNano(),
		"level":    ll.String(),
		"message":  m,
	}
	if l.a.DebugMode {
		_, fn, l, _ := runtime.Caller(2)
		fs["caller"] = fmt.Sprintf("%s:%d", fn, l)
	}

	for _, e := range es {
		for k, v := range e {
			fs[k] = v
		}
	}

	indent := ""
	if l.a.DebugMode {
		indent = "\t"
	}

	b, err := json.MarshalIndent(fs, "", indent)
	if err != nil {
		s := ""
		if l.a.DebugMode {
			s = fmt.Sprintf("{\n\t\"logger_error\": %q\n}", err)
		} else {
			s = fmt.Sprintf("{\"logger_error\":%q}", err)
		}

		b = []byte(s)
	}

	l.a.LoggerOutput.Write(b)
	l.a.LoggerOutput.Write([]byte{'\n'})
}

// LoggerLevel is the level of the logger.
type LoggerLevel uint8

// The logger levels.
const (
	// LoggerLevelDebug defines the debug level of the logger.
	LoggerLevelDebug LoggerLevel = iota

	// LoggerLevelInfo defines the info level of the logger.
	LoggerLevelInfo

	// LoggerLevelWarn defines the warn level of the logger.
	LoggerLevelWarn

	// LoggerLevelError defines the error level of the logger.
	LoggerLevelError

	// LoggerLevelFatal defines the fatal level of the logger.
	LoggerLevelFatal

	// LoggerLevelPanic defines the panic level of the logger.
	LoggerLevelPanic

	// LoggerLevelOff defines the off level of the logger. It will turn off
	// the logger.
	LoggerLevelOff
)

// String returns the string value of the ll.
func (ll LoggerLevel) String() string {
	switch ll {
	case LoggerLevelDebug:
		return "debug"
	case LoggerLevelInfo:
		return "info"
	case LoggerLevelWarn:
		return "warn"
	case LoggerLevelError:
		return "error"
	case LoggerLevelFatal:
		return "fatal"
	case LoggerLevelPanic:
		return "panic"
	}

	return "off"
}

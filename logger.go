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
	mutex *sync.Mutex
}

// theLogger is the singleton of the `logger`.
var theLogger = &logger{
	mutex: &sync.Mutex{},
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
	case LoggerLevelOff:
		return "off"
	}

	return ""
}

// log logs the m at the ll with the optional es.
func (l *logger) log(ll LoggerLevel, m string, es ...map[string]interface{}) {
	if ll < LoggerLowestLevel {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	fs := map[string]interface{}{
		"app_name": AppName,
		"time":     time.Now().UnixNano(),
		"level":    ll.String(),
		"message":  m,
	}
	if DebugMode {
		_, fn, l, _ := runtime.Caller(2)
		fs["caller"] = fmt.Sprintf("%s:%d", fn, l)
	}

	for _, e := range es {
		for k, v := range e {
			fs[k] = v
		}
	}

	var (
		b   []byte
		err error
	)

	if DebugMode {
		b, err = json.MarshalIndent(fs, "", "\t")
	} else {
		b, err = json.Marshal(fs)
	}

	if err != nil {
		if DebugMode {
			b = []byte(fmt.Sprintf(
				"{\n\t\"logger_error\": \"%v\"\n}",
				err,
			))
		} else {
			b = []byte(fmt.Sprintf(`{"logger_error":"%v"}`, err))
		}
	}

	LoggerOutput.Write(append(b, '\n'))
}

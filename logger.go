package air

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"text/template"
	"time"
)

// logger is an active logging object that generates lines of output.
type logger struct {
	template *template.Template
	once     *sync.Once
}

// loggerSingleton is the singleton of the `logger`.
var loggerSingleton = &logger{
	template: template.New("logger"),
	once:     &sync.Once{},
}

// log logs the v at the level.
func (l *logger) log(level string, v ...interface{}) {
	if !LoggerEnabled {
		return
	}

	l.once.Do(func() {
		template.Must(l.template.Parse(LoggerFormat))
	})

	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return
	}

	m := map[string]interface{}{}
	m["AppName"] = AppName
	m["Time"] = time.Now().UTC().Format(time.RFC3339)
	m["Level"] = level
	m["File"] = file
	m["Line"] = line
	m["Message"] = fmt.Sprint(v...)

	buf := &bytes.Buffer{}
	if err := l.template.Execute(buf, m); err != nil {
		return
	}

	LoggerOutput.Write(buf.Bytes())
}

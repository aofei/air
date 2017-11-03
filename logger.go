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
	mutex    *sync.Mutex
}

// loggerSingleton is the singleton of the `logger`.
var loggerSingleton = &logger{
	mutex: &sync.Mutex{},
}

// log logs the v at the level.
func (l *logger) log(level string, v ...interface{}) {
	if !LoggerEnabled {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.template == nil {
		l.template = template.Must(
			template.New("logger").Parse(LoggerFormat),
		)
	}

	_, file, line, ok := runtime.Caller(4)
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

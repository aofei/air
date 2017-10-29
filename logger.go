package air

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"text/template"
	"time"
)

// logger is used to log information generated in the runtime.
type logger struct {
	mutex    *sync.Mutex
	template *template.Template
}

// loggerSingleton is the singleton instance of the `logger`.
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

	_, file, line, _ := runtime.Caller(3)

	m := map[string]interface{}{}
	m["AppName"] = AppName
	m["Time"] = time.Now().UTC().Format(time.RFC3339)
	m["Level"] = level
	m["File"] = file
	m["Line"] = strconv.Itoa(line)
	m["Message"] = fmt.Sprint(v...)

	buf := &bytes.Buffer{}
	if err := l.template.Execute(buf, m); err == nil {
		buf.WriteByte('\n')
		LoggerOutput.Write(buf.Bytes())
	}
}

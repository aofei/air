package air

import (
	"bytes"
	"fmt"
	"path"
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

	values := map[string]interface{}{}
	values["app_name"] = AppName
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
		LoggerOutput.Write(buf.Bytes())
	}
}

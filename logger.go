package air

import (
	"encoding/json"
	"fmt"
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

// log logs the msg at the lvl with the optional extras.
func (l *logger) log(lvl, msg string, extras ...map[string]interface{}) {
	if !LoggerEnabled {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	fields := map[string]interface{}{
		"app_name": AppName,
		"time":     time.Now().UnixNano(),
		"level":    lvl,
		"message":  msg,
	}

	for _, extra := range extras {
		for k, v := range extra {
			fields[k] = v
		}
	}

	b, err := json.Marshal(fields)
	if err != nil {
		b = []byte(fmt.Sprintf(`{"error":"%v"}`, err))
	}

	LoggerOutput.Write(append(b, '\n'))
}

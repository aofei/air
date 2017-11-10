package air

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	assert.NotNil(t, loggerSingleton)
	assert.NotNil(t, loggerSingleton.template)
	assert.NotNil(t, loggerSingleton.once)

	buf := &bytes.Buffer{}
	LoggerOutput = buf

	loggerSingleton.log("INFO", "foo", "bar")
	assert.Zero(t, buf.Len())

	LoggerEnabled = true

	loggerSingleton.log("INFO", "foo", "bar")

	m := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "foobar", m["message"])

	LoggerEnabled = false
	LoggerOutput = os.Stdout
	loggerSingleton.once = &sync.Once{}
}

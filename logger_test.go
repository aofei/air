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
	assert.NotNil(t, theLogger)
	assert.NotNil(t, theLogger.template)
	assert.NotNil(t, theLogger.once)

	buf := &bytes.Buffer{}
	LoggerOutput = buf

	theLogger.log("INFO", "foo", "bar")
	assert.Zero(t, buf.Len())

	LoggerEnabled = true

	theLogger.log("INFO", "foo", "bar")

	m := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "foobar", m["message"])

	LoggerEnabled = false
	LoggerOutput = os.Stdout
	theLogger.once = &sync.Once{}
}

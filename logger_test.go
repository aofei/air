package air

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	assert.NotNil(t, theLogger)
	assert.NotNil(t, theLogger.mutex)

	buf := bytes.Buffer{}
	LoggerOutput = &buf

	theLogger.log("info", "foobar")
	assert.Zero(t, buf.Len())

	LoggerEnabled = true

	theLogger.log("info", "foo", map[string]interface{}{
		"bar": "foobar",
	})

	m := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "foo", m["message"])
	assert.Equal(t, "foobar", m["bar"])

	LoggerEnabled = false
	LoggerOutput = os.Stdout
}

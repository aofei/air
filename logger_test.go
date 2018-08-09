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

	LoggerLowestLevel = LoggerLevelOff

	buf := bytes.Buffer{}
	LoggerOutput = &buf

	theLogger.log(LoggerLevelInfo, "foobar")
	assert.Zero(t, buf.Len())

	LoggerLowestLevel = LoggerLevelInfo

	theLogger.log(LoggerLevelInfo, "foo", map[string]interface{}{
		"bar": "foobar",
	})

	m := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "foo", m["message"])
	assert.Equal(t, "foobar", m["bar"])

	LoggerLowestLevel = LoggerLevelDebug
	LoggerOutput = os.Stdout
}

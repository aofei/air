package air

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	a := New()
	l := a.logger

	assert.NotNil(t, l)
	assert.NotNil(t, l.a)
}

func TestLoggerLog(t *testing.T) {
	a := New()
	l := a.logger

	buf := bytes.Buffer{}
	a.LoggerOutput = &buf

	a.LoggerLowestLevel = LoggerLevelDebug

	buf.Reset()
	l.log(LoggerLevelDebug, "")
	assert.NotEmpty(t, buf.String())

	a.LoggerLowestLevel = LoggerLevelInfo

	buf.Reset()
	l.log(LoggerLevelDebug, "")
	assert.Empty(t, buf.String())

	a.DebugMode = true

	buf.Reset()
	l.log(LoggerLevelDebug, "", map[string]interface{}{
		"foo": "bar",
	})
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "\t")
	assert.Contains(t, buf.String(), "\n")
	assert.Contains(t, buf.String(), "\"foo\": \"bar\"")
}

func TestLoggerLevelString(t *testing.T) {
	assert.Equal(t, "debug", LoggerLevelDebug.String())
	assert.Equal(t, "info", LoggerLevelInfo.String())
	assert.Equal(t, "warn", LoggerLevelWarn.String())
	assert.Equal(t, "error", LoggerLevelError.String())
	assert.Equal(t, "fatal", LoggerLevelFatal.String())
	assert.Equal(t, "panic", LoggerLevelPanic.String())
	assert.Equal(t, "off", LoggerLevelOff.String())
	assert.Equal(t, "off", LoggerLevel(255).String())
}

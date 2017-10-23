package air

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerLoggingMethods(t *testing.T) {
	a := New()
	a.LoggerEnabled = true
	l := a.Logger
	b := &bytes.Buffer{}

	assert.Equal(t, os.Stdout, l.Output)

	l.Output = b

	log := "foobar"

	l.Debug(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Info(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Warn(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Error(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	assert.Panics(t, func() { l.Panic(log) })
	assert.NotEmpty(t, b.String())
}

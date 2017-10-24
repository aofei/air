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

	l.DEBUG(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.INFO(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.WARN(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.ERROR(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	assert.Panics(t, func() { l.PANIC(log) })
	assert.NotEmpty(t, b.String())
}

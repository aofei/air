package air

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerLoggingMethods(t *testing.T) {
	a := New()
	a.Config.LogEnabled = true
	l := a.Logger.(*logger)
	b := &bytes.Buffer{}

	assert.Equal(t, os.Stdout, l.Output())

	l.SetOutput(b)

	log := struct {
		Name   string
		Author string
	}{
		Name:   "Air",
		Author: "Aofei Sheng",
	}

	format := "%s by %s."

	m := Map{
		"Name":   log.Name,
		"Author": log.Author,
	}

	l.Print(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Printf(format, log.Name, log.Author)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Printj(m)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Debug(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Debugf(format, log.Name, log.Author)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Debugj(m)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Info(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Infof(format, log.Name, log.Author)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Infoj(m)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Warn(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Warnf(format, log.Name, log.Author)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Warnj(m)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Error(log)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Errorf(format, log.Name, log.Author)
	assert.NotEmpty(t, b.String())

	b.Reset()

	l.Errorj(m)
	assert.NotEmpty(t, b.String())

	b.Reset()

	assert.Panics(t, func() { l.Fatal(log) })
	assert.Empty(t, b.String())

	b.Reset()
	l.mutex.Unlock()

	assert.Panics(t, func() { l.Fatalf(format, log.Name, log.Author) })
	assert.Empty(t, b.String())

	b.Reset()
	l.mutex.Unlock()

	assert.Panics(t, func() { l.Fatalj(m) })
	assert.Empty(t, b.String())
}

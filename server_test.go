package air

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	a := New()
	s := a.server

	assert.NotNil(t, s.air)
	assert.NotNil(t, s.server)
}

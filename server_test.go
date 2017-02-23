package air

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerMethodAllowed(t *testing.T) {
	for _, m := range methods {
		assert.True(t, methodAllowed(m))
	}

	others := []string{
		"HEAD",
		"PATCH",
		"CONNECT",
		"OPTIONS",
		"TRACE",
	}

	for _, m := range others {
		assert.False(t, methodAllowed(m))
	}
}

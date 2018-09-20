package air

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	r := &Request{
		Method: "GET",
		Params: map[string]*RequestParam{
			"Foobar": &RequestParam{
				Name: "Foobar",
				Values: []*RequestParamValue{
					&RequestParamValue{
						i: "Foobar",
					},
				},
			},
		},
	}

	var s struct {
		Foobar string
	}

	assert.NoError(t, r.Bind(&s))
	assert.Equal(t, "Foobar", s.Foobar)
}

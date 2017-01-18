package air

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContextSetCancel(t *testing.T) {
	c := &Context{}
	c.Context = context.Background()

	assert.Nil(t, c.Done())
	c.SetCancel()
	assert.NotNil(t, c.Done())
}

func TestContextSetDeadline(t *testing.T) {
	c := &Context{}
	c.Context = context.Background()

	assert.Nil(t, c.Done())
	c.SetDeadline(time.Now())
	assert.NotNil(t, c.Done())
}

func TestContextSetTimeout(t *testing.T) {
	c := &Context{}
	c.Context = context.Background()

	assert.Nil(t, c.Done())
	c.SetTimeout(time.Nanosecond)
	assert.NotNil(t, c.Done())
}

func TestContextSetValue(t *testing.T) {
	c := &Context{}
	c.Context = context.Background()

	c.SetValue("name", "Air")
	c.SetValue("author", "Aofei Sheng")
	assert.Equal(t, "Air", c.Value("name").(string))
	assert.Equal(t, "Aofei Sheng", c.Value("author").(string))
}

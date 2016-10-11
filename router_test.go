package air

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouterMatchStatic(t *testing.T) {
	a := New()
	r := a.router
	c := a.Pool.Context()

	path := "/foo/bar.jpg"
	r.add(GET, path, func(c *Context) error {
		c.SetValue("path", path)
		return nil
	})
	r.route(GET, path, c)
	c.Handler(c)
	assert.Equal(t, path, c.Value("path"))
}

func TestRouterMatchParam(t *testing.T) {
	a := New()
	r := a.router
	c := a.Pool.Context()

	r.add(GET, "/users/:id", func(c *Context) error {
		return nil
	})
	r.route(GET, "/users/1", c)
	assert.Equal(t, "id", c.ParamNames[0])
	assert.Equal(t, "1", c.ParamValues[0])
	assert.Equal(t, "1", c.Params["id"])

	c.reset()

	r.add(GET, "/users/:uid/posts/:pid/:anchor", func(*Context) error {
		return nil
	})
	r.route(GET, "/users/1/posts/1/stars", c)
	assert.Equal(t, "uid", c.ParamNames[0])
	assert.Equal(t, "pid", c.ParamNames[1])
	assert.Equal(t, "anchor", c.ParamNames[2])
	assert.Equal(t, "1", c.ParamValues[0])
	assert.Equal(t, "1", c.ParamValues[1])
	assert.Equal(t, "stars", c.ParamValues[2])
	assert.Equal(t, "1", c.Params["uid"])
	assert.Equal(t, "1", c.Params["pid"])
	assert.Equal(t, "stars", c.Params["anchor"])
}

func TestRouterMatchAny(t *testing.T) {
	a := New()
	r := a.router
	c := a.Pool.Context()

	r.add(GET, "/*", func(*Context) error {
		return nil
	})
	r.route(GET, "/any", c)
	assert.Equal(t, "any", c.ParamValues[0])
	assert.Equal(t, "any", c.Params["*"])

	c.reset()

	r.add(GET, "/users/*", func(*Context) error {
		return nil
	})
	r.route(GET, "/users/1", c)
	assert.Equal(t, "*", c.ParamNames[0])
	assert.Equal(t, "1", c.ParamValues[0])
	assert.Equal(t, "1", c.Params["*"])
}

func TestRouterMixMatchParamAndAny(t *testing.T) {
	a := New()
	r := a.router
	c := a.Pool.Context()

	r.add(GET, "/users/:id/*", func(c *Context) error {
		return nil
	})
	r.route(GET, "/users/1/posts", c)
	c.Handler(c)
	assert.Equal(t, "id", c.ParamNames[0])
	assert.Equal(t, "*", c.ParamNames[1])
	assert.Equal(t, "1", c.ParamValues[0])
	assert.Equal(t, "posts", c.ParamValues[1])
	assert.Equal(t, "1", c.Params["id"])
	assert.Equal(t, "posts", c.Params["*"])
}

func TestRouterMatchingPriority(t *testing.T) {
	a := New()
	r := a.router
	c := a.Pool.Context()

	r.add(GET, "/users", func(c *Context) error {
		c.SetValue("a", 1)
		return nil
	})
	r.route(GET, "/users", c)
	c.Handler(c)
	assert.Equal(t, 1, c.Value("a"))

	c.reset()

	r.add(GET, "/users/new", func(c *Context) error {
		c.SetValue("b", 2)
		return nil
	})
	r.route(GET, "/users/new", c)
	c.Handler(c)
	assert.Equal(t, 2, c.Value("b"))

	c.reset()

	r.add(GET, "/users/:id", func(c *Context) error {
		c.SetValue("c", 3)
		return nil
	})
	r.route(GET, "/users/1", c)
	c.Handler(c)
	assert.Equal(t, 3, c.Value("c"))

	c.reset()

	r.add(GET, "/users/update", func(c *Context) error {
		c.SetValue("d", 4)
		return nil
	})
	r.route(GET, "/users/update", c)
	c.Handler(c)
	assert.Equal(t, 4, c.Value("d"))

	c.reset()

	r.add(GET, "/users/delete", func(c *Context) error {
		c.SetValue("e", 5)
		return nil
	})
	r.route(GET, "/users/del", c)
	c.Handler(c)
	assert.Equal(t, 3, c.Value("c"))

	c.reset()

	r.add(GET, "/users/:id/posts", func(c *Context) error {
		c.SetValue("f", 6)
		return nil
	})
	r.route(GET, "/users/1/posts", c)
	c.Handler(c)
	assert.Equal(t, 6, c.Value("f"))

	c.reset()

	r.add(GET, "/users/*", func(c *Context) error {
		c.SetValue("g", 7)
		return nil
	})
	r.route(GET, "/users/1/posts", c)
	c.Handler(c)
	assert.Equal(t, 6, c.Value("f"))

	c.reset()

	r.add(GET, "/users/*", func(c *Context) error {
		c.SetValue("h", 7)
		return nil
	})
	r.route(GET, "/users/1/followers", c)
	c.Handler(c)
	assert.Equal(t, 7, c.Value("h"))
	assert.Equal(t, "1/followers", c.Params["*"])
}

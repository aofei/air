package air

import "testing"

func TestGroupRESTfulMethods(t *testing.T) {
	a := New()
	g := NewGroup("/group", a)
	h := func(*Context) error { return nil }

	g.GET("/", h)
	g.POST("/", h)
	g.PUT("/", h)
	g.DELETE("/", h)
}

func TestGroupOtherMethods(t *testing.T) {
	a := New()
	g := NewGroup("/group", a)

	g.Static("/static", "./")
	g.File("/file", "README.md")
}

// TODO: Implement this
func TestGroupRouteGas(t *testing.T) {
}

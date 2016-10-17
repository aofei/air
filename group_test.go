package air

import "testing"

func TestGroupRESTfulMethods(t *testing.T) {
	a := New()
	g := NewGroup("/group", a)
	h := func(*Context) error { return nil }

	g.Get("/", h)
	g.Post("/", h)
	g.Put("/", h)
	g.Delete("/", h)
}

func TestGroupOtherMethods(t *testing.T) {
	a := New()
	g := NewGroup("/group", a)
	h := func(*Context) error { return nil }

	g.Any("/", h)
	g.Static("/static", "./")
	g.File("/file", "README.md")
}

// TODO: Implement this
func TestGroupRouteGas(t *testing.T) {
}

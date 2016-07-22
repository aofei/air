package air

type (
	// Group is a set of sub-routes for a specified route. It can be used for inner
	// routes that share a common middlware or functionality that should be separate
	// from the parent air instance while still inheriting from it.
	Group struct {
		prefix string
		gas    []GasFunc
		air    *Air
	}
)

// Use implements `Air#Use()` for sub-routes within the Group.
func (g *Group) Use(m ...GasFunc) {
	g.gas = append(g.gas, m...)
	// Allow all requests to reach the group as they might get dropped if router
	// doesn't find a match, making none of the group gas process.
	g.air.Any(g.prefix+"*", func(c Context) error {
		return ErrNotFound
	}, g.gas...)
}

// GET implements `Air#GET()` for sub-routes within the Group.
func (g *Group) GET(path string, h HandlerFunc, m ...GasFunc) {
	g.add(GET, path, h, m...)
}

// POST implements `Air#POST()` for sub-routes within the Group.
func (g *Group) POST(path string, h HandlerFunc, m ...GasFunc) {
	g.add(POST, path, h, m...)
}

// PUT implements `Air#PUT()` for sub-routes within the Group.
func (g *Group) PUT(path string, h HandlerFunc, m ...GasFunc) {
	g.add(PUT, path, h, m...)
}

// DELETE implements `Air#DELETE()` for sub-routes within the Group.
func (g *Group) DELETE(path string, h HandlerFunc, m ...GasFunc) {
	g.add(DELETE, path, h, m...)
}

// Any implements `Air#Any()` for sub-routes within the Group.
func (g *Group) Any(path string, handler HandlerFunc, gas ...GasFunc) {
	for _, m := range methods {
		g.add(m, path, handler, gas...)
	}
}

// Match implements `Air#Match()` for sub-routes within the Group.
func (g *Group) Match(methods []string, path string, handler HandlerFunc, gas ...GasFunc) {
	for _, m := range methods {
		g.add(m, path, handler, gas...)
	}
}

// Group creates a new sub-group with prefix and optional sub-group-level gas.
func (g *Group) Group(prefix string, gas ...GasFunc) *Group {
	m := []GasFunc{}
	m = append(m, g.gas...)
	m = append(m, gas...)
	return g.air.Group(g.prefix+prefix, m...)
}

// Static implements `Air#Static()` for sub-routes within the Group.
func (g *Group) Static(prefix, root string) {
	g.air.Static(g.prefix+prefix, root)
}

// File implements `Air#File()` for sub-routes within the Group.
func (g *Group) File(path, file string) {
	g.air.File(g.prefix+path, file)
}

func (g *Group) add(method, path string, handler HandlerFunc, gas ...GasFunc) {
	// Combine into a new slice, to avoid accidentally passing the same
	// slice for multiple routes, which would lead to later add() calls overwriting
	// the gas from earlier calls
	m := []GasFunc{}
	m = append(m, g.gas...)
	m = append(m, gas...)
	g.air.add(method, g.prefix+path, handler, m...)
}

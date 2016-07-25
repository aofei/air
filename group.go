package air

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share a common gas or functionality that should be separate
// from the parent air instance while still inheriting from it.
type Group struct {
	prefix string
	gases  []GasFunc
	air    *Air
}

// Contain implements `Air#Contain()` for sub-routes within the Group.
func (g *Group) Contain(gases ...GasFunc) {
	g.gases = append(g.gases, gases...)
	// Allow all requests to reach the group as they might get dropped if router
	// doesn't find a match, making none of the group gas process.
	for _, m := range methods {
		g.air.add(m, g.prefix+"*", func(c Context) error {
			return ErrNotFound
		}, g.gases...)
	}
}

// GET implements `Air#GET()` for sub-routes within the Group.
func (g *Group) GET(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(GET, path, handler, gases...)
}

// POST implements `Air#POST()` for sub-routes within the Group.
func (g *Group) POST(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(POST, path, handler, gases...)
}

// PUT implements `Air#PUT()` for sub-routes within the Group.
func (g *Group) PUT(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(PUT, path, handler, gases...)
}

// DELETE implements `Air#DELETE()` for sub-routes within the Group.
func (g *Group) DELETE(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(DELETE, path, handler, gases...)
}

// Group creates a new sub-group with prefix and optional sub-group-level gas.
func (g *Group) Group(prefix string, gases ...GasFunc) *Group {
	gs := []GasFunc{}
	gs = append(gs, g.gases...)
	gs = append(gs, gases...)
	return g.air.Group(g.prefix+prefix, gs...)
}

// Static implements `Air#Static()` for sub-routes within the Group.
func (g *Group) Static(prefix, root string) {
	g.air.Static(g.prefix+prefix, root)
}

// File implements `Air#File()` for sub-routes within the Group.
func (g *Group) File(path, file string) {
	g.air.File(g.prefix+path, file)
}

func (g *Group) add(method, path string, handler HandlerFunc, gases ...GasFunc) {
	// Combine into a new slice, to avoid accidentally passing the same
	// slice for multiple routes, which would lead to later add() calls overwriting
	// the gas from earlier calls
	gs := []GasFunc{}
	gs = append(gs, g.gases...)
	gs = append(gs, gases...)
	g.air.add(method, g.prefix+path, handler, gs...)
}

package air

import "path"

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share a common gas or functionality that should be separate
// from the parent `Air` instance while still inheriting from it.
type Group struct {
	prefix string
	gases  []GasFunc
	air    *Air
}

// NewGroup returns a new router group with prefix and optional group-level gases.
func NewGroup(prefix string, a *Air, gases ...GasFunc) *Group {
	g := &Group{prefix: prefix, air: a}
	g.Contain(gases...)
	// Allow all requests to reach the group as they might get dropped if router
	// doesn't find a match, making none of the group gas process.
	g.air.Any(g.prefix+"*", func(c *Context) error {
		return ErrNotFound
	}, g.gases...)
	return g
}

// NewSubGroup creates a new sub-group with prefix and optional sub-group-level gases.
func (g *Group) NewSubGroup(prefix string, gases ...GasFunc) *Group {
	gs := []GasFunc{}
	gs = append(gs, g.gases...)
	gs = append(gs, gases...)
	return NewGroup(g.prefix+prefix, g.air, gs...)
}

// Contain implements `Air#Contain()`.
func (g *Group) Contain(gases ...GasFunc) {
	g.gases = append(g.gases, gases...)
}

// Get implements `Air#Get()`.
func (g *Group) Get(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(GET, path, handler, gases...)
}

// Post implements `Air#Post()`.
func (g *Group) Post(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(POST, path, handler, gases...)
}

// Put implements `Air#Put()`.
func (g *Group) Put(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(PUT, path, handler, gases...)
}

// Delete implements `Air#Delete()`.
func (g *Group) Delete(path string, handler HandlerFunc, gases ...GasFunc) {
	g.add(DELETE, path, handler, gases...)
}

// Any implements `Air#Any()` for sub-routes within the Group.
func (g *Group) Any(path string, handler HandlerFunc, gases ...GasFunc) {
	for _, m := range methods {
		g.add(m, path, handler, gases...)
	}
}

// Static implements `Air#Static()`.
func (g *Group) Static(prefix, root string) {
	g.Get(g.prefix+prefix+"*", func(c *Context) error {
		return c.File(path.Join(root, c.Params[c.ParamNames[0]]))
	})
}

// File implements `Air#File()`.
func (g *Group) File(path, file string) {
	g.Get(g.prefix+path, func(c *Context) error {
		return c.File(file)
	})
}

// add implements `Air#add()`.
func (g *Group) add(method, path string, handler HandlerFunc, gases ...GasFunc) {
	// Combine into a new slice to avoid accidentally passing the same slice for
	// multiple routes, which would lead to later add() calls overwriting the
	// gas from earlier calls.
	gs := []GasFunc{}
	gs = append(gs, g.gases...)
	gs = append(gs, gases...)
	g.air.add(method, g.prefix+path, handler, gs...)
}

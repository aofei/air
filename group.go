package air

import "path"

// Group is a set of sub-routes for a specified route. It can be used for inner routes that share a
// common gas or functionality that should be separate from the parent `Air` instance while still
// inheriting from it.
type Group struct {
	prefix string
	gases  []GasFunc
	air    *Air
}

// NewGroup returns a pointer of a new router group with prefix and optional group-level gases.
func NewGroup(a *Air, prefix string, gases ...GasFunc) *Group {
	g := &Group{prefix: prefix, air: a}
	g.Contain(gases...)

	// Allow all requests to reach the group as they might get dropped if router doesn't find a
	// match, making none of the group gas process.
	path := g.prefix + "*"
	h := func(c *Context) error { return ErrNotFound }
	for _, m := range methods {
		g.air.add(m, path, h, g.gases...)
	}

	return g
}

// NewSubGroup creates a pointer of a new sub-group with prefix and optional sub-group-level gases.
func (g *Group) NewSubGroup(prefix string, gases ...GasFunc) *Group {
	return NewGroup(g.air, g.prefix+prefix, append(g.gases, gases...)...)
}

// Contain implements the `Air#Contain()`.
func (g *Group) Contain(gases ...GasFunc) {
	g.gases = append(g.gases, gases...)
}

// GET implements the `Air#GET()`.
func (g *Group) GET(path string, h HandlerFunc, gases ...GasFunc) {
	g.add(GET, path, h, gases...)
}

// POST implements the `Air#POST()`.
func (g *Group) POST(path string, h HandlerFunc, gases ...GasFunc) {
	g.add(POST, path, h, gases...)
}

// PUT implements the `Air#PUT()`.
func (g *Group) PUT(path string, h HandlerFunc, gases ...GasFunc) {
	g.add(PUT, path, h, gases...)
}

// DELETE implements the `Air#DELETE()`.
func (g *Group) DELETE(path string, h HandlerFunc, gases ...GasFunc) {
	g.add(DELETE, path, h, gases...)
}

// Static implements the `Air#Static()`.
func (g *Group) Static(prefix, root string) {
	g.GET(prefix+"*", func(c *Context) error {
		c.Data["file"] = path.Join(root, c.Params[c.ParamNames[0]])
		return c.File()
	})
}

// File implements the `Air#File()`.
func (g *Group) File(path, file string) {
	g.GET(path, func(c *Context) error {
		c.Data["file"] = file
		return c.File()
	})
}

// add implements the `Air#add()`.
func (g *Group) add(method, path string, h HandlerFunc, gases ...GasFunc) {
	if path == "/" {
		path = ""
	}
	g.air.add(method, g.prefix+path, h, append(g.gases, gases...)...)
}

package air

import "path"

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share a common gas or functionality that should be separate from
// the parent `Air` instance while still inheriting from it.
type Group struct {
	air *Air

	prefix string
	gases  []Gas
}

// NewGroup returns a pointer of a new router group with the prefix and the
// optional group-level gases.
func NewGroup(a *Air, prefix string, gases ...Gas) *Group {
	return &Group{
		air:    a,
		prefix: prefix,
		gases:  gases,
	}
}

// NewSubGroup creates a pointer of a new sub-group with the prefix and the
// optional sub-group-level gases.
func (g *Group) NewSubGroup(prefix string, gases ...Gas) *Group {
	return NewGroup(g.air, g.prefix+prefix, append(g.gases, gases...)...)
}

// Contain implements the `Air#Contain()`.
func (g *Group) Contain(gases ...Gas) {
	g.gases = append(g.gases, gases...)
}

// GET implements the `Air#GET()`.
func (g *Group) GET(path string, h Handler, gases ...Gas) {
	g.add(GET, path, h, gases...)
}

// HEAD implements the `Air#HEAD()`.
func (g *Group) HEAD(path string, h Handler, gases ...Gas) {
	g.add(HEAD, path, h, gases...)
}

// POST implements the `Air#POST()`.
func (g *Group) POST(path string, h Handler, gases ...Gas) {
	g.add(POST, path, h, gases...)
}

// PUT implements the `Air#PUT()`.
func (g *Group) PUT(path string, h Handler, gases ...Gas) {
	g.add(PUT, path, h, gases...)
}

// PATCH implements the `Air#PATCH()`.
func (g *Group) PATCH(path string, h Handler, gases ...Gas) {
	g.add(PATCH, path, h, gases...)
}

// DELETE implements the `Air#DELETE()`.
func (g *Group) DELETE(path string, h Handler, gases ...Gas) {
	g.add(DELETE, path, h, gases...)
}

// CONNECT implements the `Air#CONNECT()`.
func (g *Group) CONNECT(path string, h Handler, gases ...Gas) {
	g.add(CONNECT, path, h, gases...)
}

// OPTIONS implements the `Air#OPTIONS()`.
func (g *Group) OPTIONS(path string, h Handler, gases ...Gas) {
	g.add(OPTIONS, path, h, gases...)
}

// TRACE implements the `Air#TRACE()`.
func (g *Group) TRACE(path string, h Handler, gases ...Gas) {
	g.add(TRACE, path, h, gases...)
}

// Static implements the `Air#Static()`.
func (g *Group) Static(prefix, root string) {
	g.GET(prefix+"*", func(c *Context) error {
		return c.File(path.Join(root, c.Param("*")))
	})
}

// File implements the `Air#File()`.
func (g *Group) File(path, file string) {
	g.GET(path, func(c *Context) error {
		return c.File(file)
	})
}

// add implements the `Air#add()`.
func (g *Group) add(method, path string, h Handler, gases ...Gas) {
	if path == "/" {
		path = ""
	}
	g.air.add(method, g.prefix+path, h, append(g.gases, gases...)...)
}

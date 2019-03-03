package air

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share common gases or functionality that should be separate from
// the parent while still inheriting from it.
type Group struct {
	// Air is where the current group belong.
	Air *Air

	// Prefix is the prefix of all route paths of the current group.
	//
	// All paths of routes registered by the current group will share the
	// same prefix.
	//
	// The path prefix may consits of static component(s) and param
	// component(s). But it must not contain an any param component.
	Prefix string

	// Gases is the group-level gases of the current group.
	//
	// All gases of routes registered by the current group will share the
	// same group-level gases at the bottom of the stack.
	//
	// The gases is always FILO.
	Gases []Gas
}

// GET is just like the `Air.GET`.
func (g *Group) GET(path string, h Handler, gases ...Gas) {
	g.Air.GET(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// HEAD is just like the `Air.HEAD`.
func (g *Group) HEAD(path string, h Handler, gases ...Gas) {
	g.Air.HEAD(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// POST is just like the `Air.POST`.
func (g *Group) POST(path string, h Handler, gases ...Gas) {
	g.Air.POST(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// PUT is just like the `Air.PUT`.
func (g *Group) PUT(path string, h Handler, gases ...Gas) {
	g.Air.PUT(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// PATCH is just like the `Air.PATCH`.
func (g *Group) PATCH(path string, h Handler, gases ...Gas) {
	g.Air.PATCH(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// DELETE is just like the `Air.DELETE`.
func (g *Group) DELETE(path string, h Handler, gases ...Gas) {
	g.Air.DELETE(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// CONNECT is just like the `Air.CONNECT`.
func (g *Group) CONNECT(path string, h Handler, gases ...Gas) {
	g.Air.CONNECT(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// OPTIONS is just like the `Air.OPTIONS`.
func (g *Group) OPTIONS(path string, h Handler, gases ...Gas) {
	g.Air.OPTIONS(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// TRACE is just like the `Air.TRACE`.
func (g *Group) TRACE(path string, h Handler, gases ...Gas) {
	g.Air.TRACE(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// BATCH is just like the `Air.BATCH`.
func (g *Group) BATCH(methods []string, path string, h Handler, gases ...Gas) {
	g.Air.BATCH(methods, g.Prefix+path, h, append(g.Gases, gases...)...)
}

// FILE is just like the `Air.FILE`.
func (g *Group) FILE(path, file string, gases ...Gas) {
	g.Air.FILE(g.Prefix+path, file, append(g.Gases, gases...)...)
}

// FILES is just like the `Air.FILES`.
func (g *Group) FILES(prefix, root string, gases ...Gas) {
	g.Air.FILES(g.Prefix+prefix, root, append(g.Gases, gases...)...)
}

// Group is just like the `Air.Group`.
func (g *Group) Group(prefix string, gases ...Gas) *Group {
	return g.Air.Group(g.Prefix+prefix, append(g.Gases, gases...)...)
}

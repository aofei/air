package air

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share common gases or functionality that should be separate from
// the parent while still inheriting from it.
type Group struct {
	Prefix string
	Gases  []Gas
}

// GET implements the `GET()`.
func (g *Group) GET(path string, h Handler, gases ...Gas) {
	GET(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// HEAD implements the `HEAD()`.
func (g *Group) HEAD(path string, h Handler, gases ...Gas) {
	HEAD(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// POST implements the `POST()`.
func (g *Group) POST(path string, h Handler, gases ...Gas) {
	POST(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// PUT implements the `PUT()`.
func (g *Group) PUT(path string, h Handler, gases ...Gas) {
	PUT(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// PATCH implements the `PATCH()`.
func (g *Group) PATCH(path string, h Handler, gases ...Gas) {
	PATCH(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// DELETE implements the `DELETE()`.
func (g *Group) DELETE(path string, h Handler, gases ...Gas) {
	DELETE(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// CONNECT implements the `CONNECT()`.
func (g *Group) CONNECT(path string, h Handler, gases ...Gas) {
	CONNECT(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// OPTIONS implements the `OPTIONS()`.
func (g *Group) OPTIONS(path string, h Handler, gases ...Gas) {
	OPTIONS(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// TRACE implements the `TRACE()`.
func (g *Group) TRACE(path string, h Handler, gases ...Gas) {
	TRACE(g.Prefix+path, h, append(g.Gases, gases...)...)
}

// STATIC implements the `STATIC()`.
func (g *Group) STATIC(prefix, root string, gases ...Gas) {
	STATIC(g.Prefix+prefix, root, append(g.Gases, gases...)...)
}

// FILE implements the `FILE()`.
func (g *Group) FILE(path, file string, gases ...Gas) {
	FILE(g.Prefix+path, file, append(g.Gases, gases...)...)
}

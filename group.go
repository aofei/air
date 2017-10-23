package air

import "path"

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share common gases or functionality that should be separate from
// the parent instance while still inheriting from it.
type Group struct {
	air *Air

	Prefix string
	Gases  []Gas
}

// NewGroup returns a new instance of the `Group` with the prefix.
func NewGroup(a *Air, prefix string) *Group {
	return &Group{
		air:    a,
		Prefix: prefix,
	}
}

// GET implements the `Air#GET()`.
func (g *Group) GET(path string, h Handler, gases ...Gas) {
	g.add("GET", path, h, gases...)
}

// HEAD implements the `Air#HEAD()`.
func (g *Group) HEAD(path string, h Handler, gases ...Gas) {
	g.add("HEAD", path, h, gases...)
}

// POST implements the `Air#POST()`.
func (g *Group) POST(path string, h Handler, gases ...Gas) {
	g.add("POST", path, h, gases...)
}

// PUT implements the `Air#PUT()`.
func (g *Group) PUT(path string, h Handler, gases ...Gas) {
	g.add("PUT", path, h, gases...)
}

// PATCH implements the `Air#PATCH()`.
func (g *Group) PATCH(path string, h Handler, gases ...Gas) {
	g.add("PATCH", path, h, gases...)
}

// DELETE implements the `Air#DELETE()`.
func (g *Group) DELETE(path string, h Handler, gases ...Gas) {
	g.add("DELETE", path, h, gases...)
}

// CONNECT implements the `Air#CONNECT()`.
func (g *Group) CONNECT(path string, h Handler, gases ...Gas) {
	g.add("CONNECT", path, h, gases...)
}

// OPTIONS implements the `Air#OPTIONS()`.
func (g *Group) OPTIONS(path string, h Handler, gases ...Gas) {
	g.add("OPTIONS", path, h, gases...)
}

// TRACE implements the `Air#TRACE()`.
func (g *Group) TRACE(path string, h Handler, gases ...Gas) {
	g.add("TRACE", path, h, gases...)
}

// Static implements the `Air#Static()`.
func (g *Group) Static(prefix, root string) {
	g.GET(prefix+"*", func(req *Request, res *Response) error {
		return res.File(path.Join(root, req.PathParams["*"]))
	})
}

// File implements the `Air#File()`.
func (g *Group) File(path, file string) {
	g.GET(path, func(req *Request, res *Response) error {
		return res.File(file)
	})
}

// add implements the `Air#add()`.
func (g *Group) add(method, path string, h Handler, gases ...Gas) {
	if path == "/" {
		path = ""
	}
	g.air.add(method, g.Prefix+path, h, append(g.Gases, gases...)...)
}

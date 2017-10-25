package air

import "path"

// Group is a set of sub-routes for a specified route. It can be used for inner
// routes that share common gases or functionality that should be separate from
// the parent instance while still inheriting from it.
type Group struct {
	Air    *Air
	Prefix string
	Gases  []Gas
}

// GET implements the `Air#GET()`.
func (g *Group) GET(path string, h Handler, gases ...Gas) {
	g.register("GET", path, h, gases...)
}

// HEAD implements the `Air#HEAD()`.
func (g *Group) HEAD(path string, h Handler, gases ...Gas) {
	g.register("HEAD", path, h, gases...)
}

// POST implements the `Air#POST()`.
func (g *Group) POST(path string, h Handler, gases ...Gas) {
	g.register("POST", path, h, gases...)
}

// PUT implements the `Air#PUT()`.
func (g *Group) PUT(path string, h Handler, gases ...Gas) {
	g.register("PUT", path, h, gases...)
}

// PATCH implements the `Air#PATCH()`.
func (g *Group) PATCH(path string, h Handler, gases ...Gas) {
	g.register("PATCH", path, h, gases...)
}

// DELETE implements the `Air#DELETE()`.
func (g *Group) DELETE(path string, h Handler, gases ...Gas) {
	g.register("DELETE", path, h, gases...)
}

// CONNECT implements the `Air#CONNECT()`.
func (g *Group) CONNECT(path string, h Handler, gases ...Gas) {
	g.register("CONNECT", path, h, gases...)
}

// OPTIONS implements the `Air#OPTIONS()`.
func (g *Group) OPTIONS(path string, h Handler, gases ...Gas) {
	g.register("OPTIONS", path, h, gases...)
}

// TRACE implements the `Air#TRACE()`.
func (g *Group) TRACE(path string, h Handler, gases ...Gas) {
	g.register("TRACE", path, h, gases...)
}

// STATIC implements the `Air#STATIC()`.
func (g *Group) STATIC(prefix, root string) {
	g.GET(prefix+"*", func(req *Request, res *Response) error {
		return res.File(path.Join(root, req.PathParams["*"]))
	})
}

// FILE implements the `Air#FILE()`.
func (g *Group) FILE(path, file string) {
	g.GET(path, func(req *Request, res *Response) error {
		return res.File(file)
	})
}

// register implements the `Air#register()`.
func (g *Group) register(method, path string, h Handler, gases ...Gas) {
	if path == "/" {
		path = ""
	}
	g.Air.register(method, g.Prefix+path, h, append(g.Gases, gases...)...)
}

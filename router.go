package air

import (
	"fmt"
	"strings"
	"unsafe"
)

type (
	// router is the registry of all registered routes for an `Air` instance for the HTTP
	// request matching and the HTTP URL path params parsing.
	router struct {
		air *Air

		routes map[string]*route
		tree   *node
	}

	// route contains a handler and information for matching against the HTTP requests.
	route struct {
		method  string
		path    string
		handler string
	}

	// node is the node of the field `tree` of the `router`.
	node struct {
		kind          nodeKind
		label         byte
		prefix        string
		methodHandler *methodHandler
		parent        *node
		children      []*node
		pristinePath  string
		paramNames    []string
	}

	// nodekind is the kind of the `node`.
	nodeKind uint8

	// methodHandler is a set of the `Handler` distinguish by method.
	methodHandler struct {
		get     Handler
		head    Handler
		post    Handler
		put     Handler
		patch   Handler
		delete  Handler
		connect Handler
		options Handler
		trace   Handler
	}
)

// node kinds
const (
	staticKind nodeKind = iota
	paramKind
	anyKind
)

// newRouter returns a pointer of a new instance of the `router`.
func newRouter(a *Air) *router {
	return &router{
		air:    a,
		routes: make(map[string]*route),
		tree: &node{
			methodHandler: &methodHandler{},
		},
	}
}

// checkPath checks whether the path is valid.
func (r *router) checkPath(path string) {
	if path == "" {
		panic("air: the path cannot be empty")
	} else if path[0] != '/' {
		panic("air: the path must start with the /")
	} else if path != "/" && hasLastSlash(path) {
		panic("air: the path cannot end with the /, except the root path")
	} else if strings.Contains(path, "//") {
		panic("air: the path cannot have the //")
	} else if strings.Count(path, ":") > 1 {
		ps := strings.Split(path, "/")
		for _, p := range ps {
			if strings.Count(p, ":") > 1 {
				panic("air: adjacent params in the path must be separated by the /")
			}
		}
	} else if strings.Contains(path, "*") {
		if strings.Count(path, "*") > 1 {
			panic("air: only one * is allowed in the path")
		} else if path[len(path)-1] != '*' {
			panic("air: the * can only appear at the end of the path")
		} else if strings.Contains(path[strings.LastIndex(path, "/"):], ":") {
			panic("air: adjacent param and the * in the path must be separated by the /")
		}
	}
}

// checkRoute checks whether the route [method path] is valid.
func (r *router) checkRoute(method, path string) {
	if r.routes[method+path] != nil {
		panic(fmt.Sprintf("air: the route [%s %s] is already registered", method, path))
	}

	for _, route := range r.routes {
		if route.method == method &&
			pathWithoutParamNames(route.path) == pathWithoutParamNames(path) {
			panic(fmt.Sprintf(
				"air: the route [%s %s] and the route [%s %s] are ambiguous",
				method, path, route.method, route.path),
			)
		}
	}
}

// add registers a new route for the method and the path with the matching h.
func (r *router) add(method, path string, h Handler) {
	// Checks
	r.checkPath(path)
	r.checkRoute(method, path)

	ppath := path        // Pristine path
	pnames := []string{} // Param names

	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(method, path[:i], nil, staticKind, "", nil)

			for ; i < l && path[i] != '/'; i++ {
			}

			pname := path[j:i]

			for _, pn := range pnames {
				if pn == pname {
					panic("air: the path cannot have duplicate param names")
				}
			}

			pnames = append(pnames, pname)
			path = path[:j] + path[i:]

			if i, l = j, len(path); i == l {
				r.insert(method, path, h, paramKind, ppath, pnames)
				return
			}

			r.insert(method, path[:i], nil, paramKind, ppath, pnames)
		} else if path[i] == '*' {
			r.insert(method, path[:i], nil, staticKind, "", nil)
			pnames = append(pnames, "*")
			r.insert(method, path[:i+1], h, anyKind, ppath, pnames)
			return
		}
	}

	r.insert(method, path, h, staticKind, ppath, pnames)
}

// insert inserts a new route into the tree of the r.
func (r *router) insert(method, path string, h Handler, k nodeKind, ppath string,
	pnames []string) {
	if l := len(pnames); l > r.air.paramCap {
		r.air.paramCap = l
	}

	cn := r.tree // Current node as root

	var (
		search = path
		nn     *node // Next node
		sl     int   // Search length
		pl     int   // Prefix length
		ll     int   // LCP length
		max    int   // Max number of sl and pl
	)

	for {
		sl = len(search)
		pl = len(cn.prefix)
		ll = 0

		max = pl
		if sl < max {
			max = sl
		}

		for ; ll < max && search[ll] == cn.prefix[ll]; ll++ {
		}

		if ll == 0 {
			// At root node
			cn.label = search[0]
			cn.prefix = search
			if h != nil {
				cn.kind = k
				cn.addHandler(method, h)
				cn.pristinePath = ppath
				cn.paramNames = pnames
			}
		} else if ll < pl {
			// Split node
			nn = newNode(cn.kind, cn.prefix[ll:], cn.methodHandler, cn, cn.children,
				cn.pristinePath, cn.paramNames)

			// Reset parent node
			cn.kind = staticKind
			cn.label = cn.prefix[0]
			cn.prefix = cn.prefix[:ll]
			cn.children = nil
			cn.methodHandler = &methodHandler{}
			cn.pristinePath = ""
			cn.paramNames = nil

			cn.addChild(nn)

			if ll == sl {
				// At parent node
				cn.kind = k
				cn.addHandler(method, h)
				cn.pristinePath = ppath
				cn.paramNames = pnames
			} else {
				// Create child node
				nn = newNode(k, search[ll:], &methodHandler{}, cn, nil, ppath,
					pnames)
				nn.addHandler(method, h)
				cn.addChild(nn)
			}
		} else if ll < sl {
			search = search[ll:]

			if nn = cn.childByLabel(search[0]); nn != nil {
				// Go deeper
				cn = nn
				continue
			}

			// Create child node
			nn = newNode(k, search, &methodHandler{}, cn, nil, ppath, pnames)
			nn.addHandler(method, h)
			cn.addChild(nn)
		} else if h != nil {
			// Node already exists
			cn.addHandler(method, h)
			cn.pristinePath = ppath
			cn.paramNames = pnames
		}

		return
	}
}

// route routes a handler registered for the method and the path. It also parses the HTTP URL for
// the path params and load them into the c.
func (r *router) route(method, path string, c *Context) {
	cn := r.tree // Current node as root

	var (
		search = pathClean(path)
		nn     *node    // Next node
		nk     nodeKind // Next kind
		sn     *node    // Saved node
		ss     string   // Saved search
		sl     int      // Search length
		pl     int      // Prefix length
		ll     int      // LCP length
		max    int      // Max number of sl and pl
		si     int      // Start index
	)

	// Search order: static > param > any
	for {
		if search == "" {
			break
		}

		pl = 0
		ll = 0

		if cn.label != ':' {
			sl = len(search)
			pl = len(cn.prefix)

			max = pl
			if sl < max {
				max = sl
			}

			for ; ll < max && search[ll] == cn.prefix[ll]; ll++ {
			}
		}

		if ll != pl {
			goto Struggle
		}

		if search = search[ll:]; search == "" {
			break
		}

		// Static node
		if nn = cn.child(search[0], staticKind); nn != nil {
			// Save next
			if hasLastSlash(cn.prefix) {
				nk = paramKind
				sn = cn
				ss = search
			}

			cn = nn

			continue
		}

		// Param node
	Param:
		if nn = cn.childByKind(paramKind); nn != nil {
			// Save next
			if hasLastSlash(cn.prefix) {
				nk = anyKind
				sn = cn
				ss = search
			}

			cn = nn

			for si = 0; si < len(search) && search[si] != '/'; si++ {
			}

			c.ParamValues = append(c.ParamValues, unescape(search[:si]))
			search = search[si:]

			continue
		}

		// Any node
	Any:
		if cn = cn.childByKind(anyKind); cn != nil {
			if hasLastSlash(path) {
				for si = len(path) - 1; si > 0 && path[si] == '/'; si-- {
				}
				search += path[si+1:]
			}

			if len(c.ParamValues) < len(cn.paramNames) {
				c.ParamValues = append(c.ParamValues, unescape(search))
			} else {
				c.ParamValues[len(cn.paramNames)-1] = unescape(search)
			}

			break
		}

		// Struggle for the former node
	Struggle:
		if sn != nil {
			cn = sn
			sn = nil
			search = ss

			switch nk {
			case paramKind:
				goto Param
			case anyKind:
				goto Any
			}
		}

		return
	}

	if c.Handler = cn.handler(method); c.Handler != nil {
		c.PristinePath = cn.pristinePath
		c.ParamNames = cn.paramNames
	} else {
		c.Handler = cn.checkMethodNotAllowed()
	}

	return
}

// hasLastSlash reports whether the s has the last '/'.
func hasLastSlash(s string) bool {
	return len(s) > 0 && s[len(s)-1] == '/'
}

// pathWithoutParamNames returns a path from the p without the param names.
func pathWithoutParamNames(p string) string {
	for i, l := 0, len(p); i < l; i++ {
		if p[i] == ':' {
			j := i + 1

			for ; i < l && p[i] != '/'; i++ {
			}

			p = p[:j] + p[i:]
			i, l = j, len(p)

			if i == l {
				break
			}
		}
	}
	return p
}

// pathClean returns a clean path from the p.
func pathClean(p string) string {
	if p == "" {
		return "/"
	}

	b := make([]byte, 0, len(p))

	i, l := 0, len(p)
	if p[0] == '/' {
		i = 1
	}

	for i < l {
		if p[i] == '/' {
			i++
		} else {
			b = append(b, '/')
			for ; i < l && p[i] != '/'; i++ {
				b = append(b, p[i])
			}
		}
	}

	return *(*string)(unsafe.Pointer(&b))
}

// unescape return a normal string unescaped from the s.
func unescape(s string) string {
	// Count the %, check that they're well-formed.
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			n++
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				s = s[i:]
				if len(s) > 3 {
					s = s[:3]
				}
				return ""
			}
			i += 2
		}
	}

	if n == 0 {
		return s
	}

	t := make([]byte, len(s)-2*n)
	for i, j := 0, 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			t[j] = unhex(s[i+1])<<4 | unhex(s[i+2])
			j++
			i += 2
		case '+':
			t[j] = ' '
			j++
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

// ishex reports whether the c is hex.
func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// unhex returns the normal byte from the hex char c.
func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// newNode returns a pointer of a new instance of the `node` with the provided values.
func newNode(k nodeKind, pre string, mh *methodHandler, p *node, c []*node, ppath string,
	pnames []string) *node {
	return &node{
		kind:          k,
		label:         pre[0],
		prefix:        pre,
		methodHandler: mh,
		parent:        p,
		children:      c,
		pristinePath:  ppath,
		paramNames:    pnames,
	}
}

// child returns a child `node` of the n by the provided label l and the kind t.
func (n *node) child(l byte, t nodeKind) *node {
	for _, c := range n.children {
		if c.label == l && c.kind == t {
			return c
		}
	}
	return nil
}

// childByLabel returns a child `node` of the n by the provided label l.
func (n *node) childByLabel(l byte) *node {
	for _, c := range n.children {
		if c.label == l {
			return c
		}
	}
	return nil
}

// childByKind returns a child `node` of the n by the provided kind t.
func (n *node) childByKind(t nodeKind) *node {
	for _, c := range n.children {
		if c.kind == t {
			return c
		}
	}
	return nil
}

// addChild adds the c into the children nodes of the n.
func (n *node) addChild(c *node) {
	n.children = append(n.children, c)
}

// handler returns a `Handler` by the provided method.
func (n *node) handler(method string) Handler {
	switch method {
	case GET:
		return n.methodHandler.get
	case HEAD:
		return n.methodHandler.head
	case POST:
		return n.methodHandler.post
	case PUT:
		return n.methodHandler.put
	case PATCH:
		return n.methodHandler.patch
	case DELETE:
		return n.methodHandler.delete
	case CONNECT:
		return n.methodHandler.connect
	case OPTIONS:
		return n.methodHandler.options
	case TRACE:
		return n.methodHandler.trace
	}
	return nil
}

// addHandler adds the h into the filed `methodHandler` of the n with the provided method.
func (n *node) addHandler(method string, h Handler) {
	switch method {
	case GET:
		n.methodHandler.get = h
	case HEAD:
		n.methodHandler.head = h
	case POST:
		n.methodHandler.post = h
	case PUT:
		n.methodHandler.put = h
	case PATCH:
		n.methodHandler.patch = h
	case DELETE:
		n.methodHandler.delete = h
	case CONNECT:
		n.methodHandler.connect = h
	case OPTIONS:
		n.methodHandler.options = h
	case TRACE:
		n.methodHandler.trace = h
	}
}

// checkMethodNotAllowed returns a `Handler` by checked methods.
func (n *node) checkMethodNotAllowed() Handler {
	for _, m := range methods {
		if h := n.handler(m); h != nil {
			return MethodNotAllowedHandler
		}
	}
	return NotFoundHandler
}

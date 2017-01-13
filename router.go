package air

import (
	"fmt"
	"strings"
)

type (
	// router is the registry of all registered routes for an `Air` instance for request
	// matching and URL path parameter parsing.
	router struct {
		routes map[string]*route
		tree   *node
		air    *Air
	}

	// route contains a handler and information for matching against requests.
	route struct {
		method  string
		path    string
		handler string
	}

	// node is the node of the router's tree.
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

	// nodekind is the kind of `node`.
	nodeKind uint8

	// methodHandler is a set of `HandlerFunc` distinguish by method.
	methodHandler struct {
		get    HandlerFunc
		post   HandlerFunc
		put    HandlerFunc
		delete HandlerFunc
	}
)

// node kinds
const (
	staticKind nodeKind = iota
	paramKind
	anyKind
)

// newRouter returns a pointer of a new router instance.
func newRouter(a *Air) *router {
	return &router{
		routes: make(map[string]*route),
		tree: &node{
			methodHandler: &methodHandler{},
		},
		air: a,
	}
}

// checkPath checks whether the path is valid.
func (r *router) checkPath(path string) {
	if path == "" {
		panic("path cannot be empty")
	} else if path[0] != '/' {
		panic("path must start with /")
	} else if path != "/" && path[len(path)-1] == '/' {
		panic("path cannot end with /, except the root path")
	} else if strings.Count(path, ":") > 1 {
		ps := strings.Split(path, "/")
		for _, p := range ps {
			if strings.Count(p, ":") > 1 {
				panic("adjacent params in a path must be separated by /")
			}
		}
	} else if strings.Contains(path, "*") {
		if strings.Count(path, "*") > 1 {
			panic("only one * is allowed in a path")
		} else if path[len(path)-1] != '*' {
			panic("* can only appear at the end of a path")
		} else if strings.Contains(path[strings.LastIndex(path, "/"):], ":") {
			panic("adjacent param and * in a path must be separated by /")
		}
	}
}

// checkRoute checks whether the [method path] route is valid.
func (r *router) checkRoute(method, path string) {
	if r.routes[method+path] != nil {
		panic(fmt.Sprintf("route [%s %s] is already registered", method, path))
	}

	for _, route := range r.routes {
		if route.method == method &&
			pathWithoutParamNames(route.path) == pathWithoutParamNames(path) {
			panic(fmt.Sprintf("route [%s %s] and [%s %s] are ambiguous",
				method, path, route.method, route.path))
		}
	}
}

// add registers a new route for method and path with matching h.
func (r *router) add(method, path string, h HandlerFunc) {
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
					panic("a path cannot have duplicate param names")
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

// insert inserts a new route into the tree of r.
func (r *router) insert(method, path string, h HandlerFunc, k nodeKind, ppath string,
	pnames []string) {
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

// route routes a handler registered for method and path. It also parses URL for path parameters and
// load them into c.
func (r *router) route(method, path string, c *Context) {
	cn := r.tree // Current node as root

	var (
		search = path
		nn     *node    // Next node
		nk     nodeKind // Next kind
		sn     *node    // Saved node
		ss     string   // Saved search
		sl     int      // Search length
		pl     int      // Prefix length
		ll     int      // LCP length
		max    int      // Max number of sl and pl
		i, l   int      // Temp vars
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

		if ll == pl {
			search = search[ll:]
		} else {
			cn = sn
			search = ss

			switch nk {
			case paramKind:
				goto Param
			case anyKind:
				goto Any
			}

			return
		}

		if search == "" {
			break
		}

		// Static node
		if nn = cn.child(search[0], staticKind); nn != nil {
			// Save next
			if cn.prefix[len(cn.prefix)-1] == '/' {
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
			if cn.prefix[len(cn.prefix)-1] == '/' {
				nk = anyKind
				sn = cn
				ss = search
			}

			cn = nn

			for i, l = 0, len(search); i < l && search[i] != '/'; i++ {
			}

			c.ParamValues = append(c.ParamValues, unescape(search[:i]))
			search = search[i:]

			continue
		}

		// Any node
	Any:
		if cn = cn.childByKind(anyKind); cn != nil {
			if len(c.ParamValues) == len(cn.paramNames) {
				c.ParamValues[len(c.ParamValues)-1] = unescape(search)
			} else {
				c.ParamValues = append(c.ParamValues, unescape(search))
			}

			break
		}

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

	c.Handler = cn.handler(method)
	if c.Handler == nil {
		c.Handler = cn.checkMethodNotAllowed()

		// Dig further for any, might have an empty value for *, e.g. serving a directory.

		if cn = cn.childByKind(anyKind); cn == nil {
			return
		}

		if h := cn.handler(method); h != nil {
			c.Handler = h
		} else {
			c.Handler = cn.checkMethodNotAllowed()
		}

		if len(c.ParamValues) == len(cn.paramNames) {
			c.ParamValues[len(c.ParamValues)-1] = ""
		} else {
			c.ParamValues = append(c.ParamValues, "")
		}
	}

	c.PristinePath = cn.pristinePath
	c.ParamNames = cn.paramNames

	for i, n := range c.ParamNames {
		c.Params[n] = c.ParamValues[i]
	}

	return
}

// pathWithoutParamNames returns the path from p without the param names.
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

// unescape return a normal string unescaped from s.
func unescape(s string) string {
	// Count %, check that they're well-formed.
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

// ishex returns true if c was hex.
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

// unhex returns normal byte from hex char c.
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

// newNode returns a pointer of a new instance of `node` with provided values.
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

// child returns a child `node` of n by provided label l and kint t.
func (n *node) child(l byte, t nodeKind) *node {
	for _, c := range n.children {
		if c.label == l && c.kind == t {
			return c
		}
	}
	return nil
}

// childByLabel returns a child `node` of n by provided label l.
func (n *node) childByLabel(l byte) *node {
	for _, c := range n.children {
		if c.label == l {
			return c
		}
	}
	return nil
}

// childByKind returns a child `node` of n by provided kint t.
func (n *node) childByKind(t nodeKind) *node {
	for _, c := range n.children {
		if c.kind == t {
			return c
		}
	}
	return nil
}

// addChild adds c into children nodes of n.
func (n *node) addChild(c *node) {
	n.children = append(n.children, c)
}

// handler returns a `HandlerFunc` by provided method.
func (n *node) handler(method string) HandlerFunc {
	switch method {
	case GET:
		return n.methodHandler.get
	case POST:
		return n.methodHandler.post
	case PUT:
		return n.methodHandler.put
	case DELETE:
		return n.methodHandler.delete
	default:
		return nil
	}
}

// addHandler adds h into methodHandlers of n with provided method.
func (n *node) addHandler(method string, h HandlerFunc) {
	switch method {
	case GET:
		n.methodHandler.get = h
	case POST:
		n.methodHandler.post = h
	case PUT:
		n.methodHandler.put = h
	case DELETE:
		n.methodHandler.delete = h
	}
}

// checkMethodNotAllowed returns a `HandlerFunc` by checked methods.
func (n *node) checkMethodNotAllowed() HandlerFunc {
	for _, m := range methods {
		if h := n.handler(m); h != nil {
			return MethodNotAllowedHandler
		}
	}
	return NotFoundHandler
}

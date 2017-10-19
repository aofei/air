package air

import (
	"fmt"
	"strings"
	"unsafe"
)

type (
	// router is the registry of all registered routes for an `Air` instance
	// for the HTTP request matching and the HTTP URL path params parsing.
	router struct {
		air *Air

		routes []*route
		tree   *node
	}

	// route contains a handler and information for matching against the
	// HTTP requests.
	route struct {
		method  string
		path    string
		handler Handler
	}

	// node is the node of the field `tree` of the `router`.
	node struct {
		kind       nodeKind
		label      byte
		prefix     string
		handlers   map[string]Handler
		parent     *node
		children   []*node
		paramNames []string
	}

	// nodekind is the kind of the `node`.
	nodeKind uint8
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
		air: a,
		tree: &node{
			handlers: map[string]Handler{},
		},
	}
}

// add registers a new route for the method and the path with the matching h.
func (r *router) add(method, path string, h Handler) {
	if path == "" {
		panic("air: the path cannot be empty")
	} else if path[0] != '/' {
		panic("air: the path must start with the /")
	} else if path != "/" && hasLastSlash(path) {
		panic("air: the path cannot end with the /, except the root " +
			"path")
	} else if strings.Contains(path, "//") {
		panic("air: the path cannot have the //")
	} else if strings.Count(path, ":") > 1 {
		ps := strings.Split(path, "/")
		for _, p := range ps {
			if strings.Count(p, ":") > 1 {
				panic("air: adjacent params in the path must " +
					"be separated by the /")
			}
		}
	} else if strings.Contains(path, "*") {
		if strings.Count(path, "*") > 1 {
			panic("air: only one * is allowed in the path")
		} else if path[len(path)-1] != '*' {
			panic("air: the * can only appear at the end of the " +
				"path")
		} else if strings.Contains(
			path[strings.LastIndex(path, "/"):],
			":",
		) {
			panic("air: adjacent param and the * in the path " +
				"must be separated by the /")
		}
	} else {
		for _, route := range r.routes {
			if route.method == method {
				if route.path == path {
					panic(fmt.Sprintf(
						"air: the route [%s %s] is "+
							"already registered",
						method,
						path,
					))
				} else if pathWithoutParamNames(route.path) ==
					pathWithoutParamNames(path) {
					panic(fmt.Sprintf(
						"air: the route [%s %s] and "+
							"the route [%s %s] "+
							"are ambiguous",
						method,
						path,
						route.method,
						route.path,
					))
				}
			}
		}
	}

	r.routes = append(r.routes, &route{
		method:  method,
		path:    path,
		handler: h,
	})

	paramNames := []string{}

	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(method, path[:i], nil, staticKind, nil)

			for ; i < l && path[i] != '/'; i++ {
			}

			paramName := path[j:i]

			for _, pn := range paramNames {
				if pn == paramName {
					panic("air: the path cannot have " +
						"duplicate param names")
				}
			}

			paramNames = append(paramNames, paramName)
			path = path[:j] + path[i:]

			if i, l = j, len(path); i == l {
				r.insert(method, path, h, paramKind, paramNames)
				return
			}

			r.insert(method, path[:i], nil, paramKind, paramNames)
		} else if path[i] == '*' {
			r.insert(method, path[:i], nil, staticKind, nil)
			paramNames = append(paramNames, "*")
			r.insert(method, path[:i+1], h, anyKind, paramNames)
			return
		}
	}

	r.insert(method, path, h, staticKind, paramNames)
}

// insert inserts a new route into the tree of the r.
func (r *router) insert(
	method,
	path string,
	h Handler,
	nk nodeKind,
	paramNames []string,
) {
	cn := r.tree // Current node as root

	var (
		s   = path // Search
		nn  *node  // Next node
		sl  int    // Search length
		pl  int    // Prefix length
		ll  int    // LCP length
		max int    // Max number of sl and pl
	)

	for {
		sl = len(s)
		pl = len(cn.prefix)
		ll = 0

		max = pl
		if sl < max {
			max = sl
		}

		for ; ll < max && s[ll] == cn.prefix[ll]; ll++ {
		}

		if ll == 0 {
			// At root node
			cn.label = s[0]
			cn.prefix = s
			if h != nil {
				cn.kind = nk
				cn.handlers[method] = h
				cn.paramNames = paramNames
			}
		} else if ll < pl {
			// Split node
			nn = &node{
				kind:       cn.kind,
				label:      cn.prefix[ll],
				prefix:     cn.prefix[ll:],
				handlers:   cn.handlers,
				parent:     cn,
				children:   cn.children,
				paramNames: cn.paramNames,
			}

			// Reset parent node
			cn.kind = staticKind
			cn.label = cn.prefix[0]
			cn.prefix = cn.prefix[:ll]
			cn.children = nil
			cn.handlers = map[string]Handler{}
			cn.paramNames = nil
			cn.children = append(cn.children, nn)

			if ll == sl {
				// At parent node
				cn.kind = nk
				cn.handlers[method] = h
				cn.paramNames = paramNames
			} else {
				// Create child node
				nn = &node{
					kind:       nk,
					label:      s[ll],
					prefix:     s[ll:],
					handlers:   map[string]Handler{},
					parent:     cn,
					paramNames: paramNames,
				}
				nn.handlers[method] = h
				cn.children = append(cn.children, nn)
			}
		} else if ll < sl {
			s = s[ll:]

			if nn = cn.childByLabel(s[0]); nn != nil {
				// Go deeper
				cn = nn
				continue
			}

			// Create child node
			nn = &node{
				kind:       nk,
				label:      s[0],
				prefix:     s,
				handlers:   map[string]Handler{},
				parent:     cn,
				paramNames: paramNames,
			}
			nn.handlers[method] = h
			cn.children = append(cn.children, nn)
		} else if h != nil {
			// Node already exists
			cn.handlers[method] = h
			cn.paramNames = paramNames
		}

		return
	}
}

// route returns a handler registered for the req.
func (r *router) route(req *Request) Handler {
	cn := r.tree // Current node as root

	var (
		s   = pathClean(req.URL.Path) // Search
		nn  *node                     // Next node
		nk  nodeKind                  // Next kind
		sn  *node                     // Saved node
		ss  string                    // Saved search
		sl  int                       // Search length
		pl  int                       // Prefix length
		ll  int                       // LCP length
		max int                       // Max number of sl and pl
		si  int                       // Start index
		pi  int                       // Param index
	)

	// Search order: static > param > any
	for {
		if s == "" {
			break
		}

		pl = 0
		ll = 0

		if cn.label != ':' {
			sl = len(s)
			pl = len(cn.prefix)

			max = pl
			if sl < max {
				max = sl
			}

			for ; ll < max && s[ll] == cn.prefix[ll]; ll++ {
			}
		}

		if ll != pl {
			goto Struggle
		}

		if s = s[ll:]; s == "" {
			break
		}

		// Static node
		if nn = cn.child(s[0], staticKind); nn != nil {
			// Save next
			if hasLastSlash(cn.prefix) {
				nk = paramKind
				sn = cn
				ss = s
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
				ss = s
			}

			cn = nn

			for si = 0; si < len(s) && s[si] != '/'; si++ {
			}

			req.PathParams[cn.paramNames[pi]] = unescape(s[:si])
			pi++
			s = s[si:]

			continue
		}

		// Any node
	Any:
		if cn = cn.childByKind(anyKind); cn != nil {
			if hasLastSlash(req.URL.Path) {
				si = len(req.URL.Path) - 1
				for ; si > 0 && req.URL.Path[si] == '/'; si-- {
				}
				s += req.URL.Path[si+1:]
			}

			req.PathParams["*"] = unescape(s)

			break
		}

		// Struggle for the former node
	Struggle:
		if sn != nil {
			cn = sn
			sn = nil
			s = ss

			switch nk {
			case paramKind:
				goto Param
			case anyKind:
				goto Any
			}
		}

		return NotFoundHandler
	}

	if handler := cn.handlers[req.Method]; handler != nil {
		return handler
	} else if len(cn.handlers) != 0 {
		return MethodNotAllowedHandler
	}

	return NotFoundHandler
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

// child returns a child `node` of the n by the provided label l and the kind t.
func (n *node) child(l byte, nk nodeKind) *node {
	for _, c := range n.children {
		if c.label == l && c.kind == nk {
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
func (n *node) childByKind(nk nodeKind) *node {
	for _, c := range n.children {
		if c.kind == nk {
			return c
		}
	}
	return nil
}

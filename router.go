package air

import (
	"strings"
	"sync"
)

// router is a registry of all registered routes.
type router struct {
	sync.Mutex

	a                    *Air
	tree                 *node
	registeredRoutes     map[string]bool
	maxRouteParams       int
	routeParamValuesPool *sync.Pool
}

// newRouter returns a new instance of the `router` with the a.
func newRouter(a *Air) *router {
	r := &router{
		a: a,
		tree: &node{
			handlers: map[string]Handler{},
		},
		registeredRoutes: map[string]bool{},
	}
	r.routeParamValuesPool = &sync.Pool{
		New: func() interface{} {
			return make([]string, r.maxRouteParams)
		},
	}

	return r
}

// register registers a new route for the method and the path with the matching
// h in the r with the optional route-level gases.
func (r *router) register(method, path string, h Handler, gases ...Gas) {
	r.Lock()
	defer r.Unlock()

	if path == "" {
		panic("air: route path cannot be empty")
	} else if path[0] != '/' {
		panic("air: route path must start with /")
	} else if strings.Contains(path, "//") {
		panic("air: route path cannot have //")
	} else if strings.Count(path, ":") > 1 {
		ps := strings.Split(path, "/")
		for _, p := range ps {
			if strings.Count(p, ":") > 1 {
				panic("air: adjacent param names in route " +
					"path must be separated by /")
				break
			}
		}
	} else if strings.Contains(path, "*") {
		if strings.Count(path, "*") > 1 {
			panic("air: only one * is allowed in route path")
		} else if path[len(path)-1] != '*' {
			panic("air: * can only appear at end of route path")
		} else if strings.Contains(
			path[strings.LastIndex(path, "/"):],
			":",
		) {
			panic("air: adjacent param name and * in route path " +
				"must be separated by /")
		}
	}

	routeName := method + path
	for i, l := len(method), len(routeName); i < l; i++ {
		if routeName[i] == ':' {
			j := i + 1

			for ; i < l && routeName[i] != '/'; i++ {
			}

			routeName = routeName[:j] + routeName[i:]
			i, l = j, len(routeName)

			if i == l {
				break
			}
		}
	}

	if r.registeredRoutes[routeName] {
		panic("air: route already exists")
	} else {
		r.registeredRoutes[routeName] = true
	}

	paramNames := []string{}
	nh := func(req *Request, res *Response) error {
		h := h
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}

		return h(req, res)
	}

	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(method, path[:i], nil, nodeKindStatic, nil)

			for ; i < l && path[i] != '/'; i++ {
			}

			paramName := path[j:i]

			for _, pn := range paramNames {
				if pn == paramName {
					panic("air: route path cannot have " +
						"duplicate param names")
				}
			}

			paramNames = append(paramNames, paramName)
			path = path[:j] + path[i:]

			if i, l = j, len(path); i == l {
				r.insert(
					method,
					path,
					nh,
					nodeKindParam,
					paramNames,
				)
				return
			}

			r.insert(
				method,
				path[:i],
				nil,
				nodeKindParam,
				paramNames,
			)
		} else if path[i] == '*' {
			r.insert(method, path[:i], nil, nodeKindStatic, nil)
			paramNames = append(paramNames, "*")
			r.insert(
				method,
				path[:i+1],
				nh,
				nodeKindAny,
				paramNames,
			)
			return
		}
	}

	r.insert(method, path, nh, nodeKindStatic, paramNames)
}

// insert inserts a new route into the `tree` of the r.
func (r *router) insert(
	method string,
	path string,
	h Handler,
	nk nodeKind,
	paramNames []string,
) {
	if l := len(paramNames); l > r.maxRouteParams {
		r.maxRouteParams = l
	}

	var (
		s  = path   // Search
		cn = r.tree // Current node
		nn *node    // Next node
		sl int      // Search length
		pl int      // Prefix length
		ll int      // LCP length
		ml int      // Minimum length of sl and pl
	)

	for {
		sl = len(s)
		pl = len(cn.prefix)
		ll = 0

		ml = pl
		if sl < ml {
			ml = sl
		}

		for ; ll < ml && s[ll] == cn.prefix[ll]; ll++ {
		}

		if ll == 0 { // At root node
			cn.label = s[0]
			cn.kind = nk
			cn.prefix = s
			cn.paramNames = paramNames
			if h != nil {
				cn.handlers[method] = h
			}
		} else if ll < pl { // Split node
			nn = &node{
				label:      cn.prefix[ll],
				kind:       cn.kind,
				prefix:     cn.prefix[ll:],
				parent:     cn,
				children:   cn.children,
				paramNames: cn.paramNames,
				handlers:   cn.handlers,
			}

			// Reset parent node.
			cn.label = cn.prefix[0]
			cn.kind = nodeKindStatic
			cn.prefix = cn.prefix[:ll]
			cn.children = []*node{nn}
			cn.paramNames = nil
			cn.handlers = map[string]Handler{}

			if ll == sl { // At parent node
				cn.kind = nk
				cn.paramNames = paramNames
				if h != nil {
					cn.handlers[method] = h
				}
			} else { // Create child node
				nn = &node{
					label:      s[ll],
					kind:       nk,
					prefix:     s[ll:],
					parent:     cn,
					paramNames: paramNames,
					handlers:   map[string]Handler{},
				}
				if h != nil {
					nn.handlers[method] = h
				}

				cn.children = append(cn.children, nn)
			}
		} else if ll < sl {
			s = s[ll:]

			if nn = cn.childByLabel(s[0]); nn != nil {
				// Go deeper.
				cn = nn
				continue
			}

			// Create child node.
			nn = &node{
				label:      s[0],
				kind:       nk,
				prefix:     s,
				handlers:   map[string]Handler{},
				parent:     cn,
				paramNames: paramNames,
			}
			if h != nil {
				nn.handlers[method] = h
			}

			cn.children = append(cn.children, nn)
		} else { // Node already exists
			if len(cn.paramNames) == 0 {
				cn.paramNames = paramNames
			}

			if h != nil {
				cn.handlers[method] = h
			}
		}

		break
	}
}

// route returns a handler registered for the req.
func (r *router) route(req *Request) Handler {
	var (
		s, _ = splitPathQuery(req.Path) // Search
		cn   = r.tree                   // Current node
		nn   *node                      // Next node
		nk   nodeKind                   // Next kind
		sn   *node                      // Saved node
		ss   string                     // Saved search
		sl   int                        // Search length
		pl   int                        // Prefix length
		ll   int                        // LCP length
		ml   int                        // Minimum length of sl and pl
		i    int                        // Index
		pi   int                        // Param index
	)

	// Search order: static > param > any.
	for {
		if s == "" {
			break
		}

		for i, sl = 1, len(s); i < sl && s[i] == '/'; i++ {
		}

		s = s[i-1:]

		pl = 0
		ll = 0

		if cn.label != ':' {
			pl = len(cn.prefix)

			ml = pl
			if sl = len(s); sl < ml {
				ml = sl
			}

			for ; ll < ml && s[ll] == cn.prefix[ll]; ll++ {
			}
		}

		if ll != pl {
			goto Struggle
		}

		if s = s[ll:]; s == "" {
			if len(cn.handlers) == 0 {
				if cn.childByKind(nodeKindParam) != nil {
					goto Param
				} else if cn.childByKind(nodeKindAny) != nil {
					goto Any
				}
			}

			break
		}

		// Static node.
		if nn = cn.child(s[0], nodeKindStatic); nn != nil {
			// Save next.
			if l := len(cn.prefix); l > 0 && cn.prefix[l-1] == '/' {
				nk = nodeKindParam
				sn = cn
				ss = s
			}

			cn = nn

			continue
		}

		// Param node.
	Param:
		if nn = cn.childByKind(nodeKindParam); nn != nil {
			// Save next.
			if l := len(cn.prefix); l > 0 && cn.prefix[l-1] == '/' {
				nk = nodeKindAny
				sn = cn
				ss = s
			}

			cn = nn

			for i, sl = 0, len(s); i < sl && s[i] != '/'; i++ {
			}

			if req.routeParamValues == nil {
				req.routeParamValues = r.routeParamValuesPool.
					Get().([]string)
			}

			req.routeParamValues[pi] = s[:i]
			pi++

			s = s[i:]

			continue
		}

		// Any node.
	Any:
		if cn = cn.childByKind(nodeKindAny); cn != nil {
			if req.routeParamValues == nil {
				req.routeParamValues = r.routeParamValuesPool.
					Get().([]string)
			}

			req.routeParamValues[len(cn.paramNames)-1] = s

			break
		}

		// Struggle for the former node.
	Struggle:
		if sn != nil {
			cn = sn
			sn = nil
			s = ss

			switch nk {
			case nodeKindParam:
				goto Param
			case nodeKindAny:
				goto Any
			}
		}

		return r.a.NotFoundHandler
	}

	if h := cn.handlers[req.Method]; h != nil {
		req.routeParamNames = cn.paramNames
		return h
	} else if len(cn.handlers) != 0 {
		return r.a.MethodNotAllowedHandler
	}

	return r.a.NotFoundHandler
}

// node is the node of the radix tree.
type node struct {
	label      byte
	kind       nodeKind
	prefix     string
	parent     *node
	children   []*node
	paramNames []string
	handlers   map[string]Handler
}

// child returns a child `node` of the n by the label and the kind.
func (n *node) child(label byte, kind nodeKind) *node {
	for _, c := range n.children {
		if c.label == label && c.kind == kind {
			return c
		}
	}

	return nil
}

// childByLabel returns a child `node` of the n by the l.
func (n *node) childByLabel(l byte) *node {
	for _, c := range n.children {
		if c.label == l {
			return c
		}
	}

	return nil
}

// childByKind returns a child `node` of the n by the k.
func (n *node) childByKind(k nodeKind) *node {
	for _, c := range n.children {
		if c.kind == k {
			return c
		}
	}

	return nil
}

// nodeKind is a kind of the `node`.
type nodeKind uint8

// The node kinds.
const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindAny
)

// splitPathQuery splits the p of the form "path?query" into path and query.
func splitPathQuery(p string) (path, query string) {
	i, l := 0, len(p)
	for ; i < l && p[i] != '?'; i++ {
	}

	if i < l {
		return p[:i], p[i+1:]
	}

	return p, ""
}

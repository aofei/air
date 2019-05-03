package air

import (
	"net/url"
	ppath "path"
	"strings"
	"sync"
)

// router is a registry of all registered routes.
type router struct {
	sync.Mutex

	a                    *Air
	routeTree            *routeNode
	registeredRoutes     map[string]bool
	maxRouteParams       int
	routeParamValuesPool *sync.Pool
}

// newRouter returns a new instance of the `router` with the a.
func newRouter(a *Air) *router {
	r := &router{
		a: a,
		routeTree: &routeNode{
			handlers: map[string]Handler{},
		},
		registeredRoutes: map[string]bool{},
	}
	r.routeParamValuesPool = &sync.Pool{
		New: func() interface{} {
			return make([]string, 0, r.maxRouteParams)
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
	} else if h == nil {
		panic("air: route handler cannot be nil")
	}

	path = ppath.Clean(path)
	path = url.PathEscape(path)
	path = strings.Replace(path, "%2F", "/", -1)
	path = strings.Replace(path, "%2A", "*", -1)
	if path[0] != '/' {
		panic("air: route path must start with /")
	} else if strings.Count(path, ":") > 1 {
		ps := strings.Split(path, "/")
		for _, p := range ps {
			if strings.Count(p, ":") > 1 {
				panic("air: adjacent param names in route " +
					"path must be separated by /")
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

	rh := func(req *Request, res *Response) error {
		h := h
		for i := len(gases) - 1; i >= 0; i-- {
			h = gases[i](h)
		}

		return h(req, res)
	}

	paramNames := []string{}
	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(
				method,
				path[:i],
				nil,
				routeNodeTypeStatic,
				nil,
			)

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
					rh,
					routeNodeTypeParam,
					paramNames,
				)
				return
			}

			r.insert(
				method,
				path[:i],
				nil,
				routeNodeTypeParam,
				paramNames,
			)
		} else if path[i] == '*' {
			r.insert(
				method,
				path[:i],
				nil,
				routeNodeTypeStatic,
				nil,
			)
			paramNames = append(paramNames, "*")
			r.insert(
				method,
				path[:i+1],
				rh,
				routeNodeTypeAny,
				paramNames,
			)
			return
		}
	}

	r.insert(method, path, rh, routeNodeTypeStatic, paramNames)
}

// insert inserts a new route into the `r.routeTree`.
func (r *router) insert(
	method string,
	path string,
	h Handler,
	nt routeNodeType,
	paramNames []string,
) {
	if l := len(paramNames); l > r.maxRouteParams {
		r.maxRouteParams = l
	}

	var (
		s  = path        // Search
		cn = r.routeTree // Current node
		nn *routeNode    // Next node
		sl int           // Search length
		pl int           // Prefix length
		ll int           // LCP length
		ml int           // Minimum length of sl and pl
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
			cn.nType = nt
			cn.prefix = s
			cn.paramNames = paramNames
			if h != nil {
				cn.handlers[method] = h
			}
		} else if ll < pl { // Split node
			nn = &routeNode{
				label:      cn.prefix[ll],
				nType:      cn.nType,
				prefix:     cn.prefix[ll:],
				children:   cn.children,
				paramNames: cn.paramNames,
				handlers:   cn.handlers,
			}

			// Reset current node.
			cn.label = cn.prefix[0]
			cn.nType = routeNodeTypeStatic
			cn.prefix = cn.prefix[:ll]
			cn.children = []*routeNode{nn}
			cn.paramNames = nil
			cn.handlers = map[string]Handler{}

			if ll == sl { // At current node
				cn.nType = nt
				cn.paramNames = paramNames
				if h != nil {
					cn.handlers[method] = h
				}
			} else { // Create child node
				nn = &routeNode{
					label:      s[ll],
					nType:      nt,
					prefix:     s[ll:],
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
			nn = &routeNode{
				label:      s[0],
				nType:      nt,
				prefix:     s,
				handlers:   map[string]Handler{},
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
		cn   = r.routeTree              // Current node
		nn   *routeNode                 // Next node
		sn   *routeNode                 // Saved node
		snt  routeNodeType              // Saved type
		ss   string                     // Saved search
		sapn *routeNode                 // Saved any parent node
		saps string                     // Saved any parent search
		sl   int                        // Search length
		pl   int                        // Prefix length
		ll   int                        // LCP length
		ml   int                        // Minimum length of sl and pl
		i    int                        // Index
		pc   int                        // Param counter
	)

	// Search order: static route > param route > any route.
	for {
		if s == "" {
			if len(cn.handlers) == 0 {
				if cn.childByType(routeNodeTypeParam) != nil {
					goto TryParam
				}

				if cn.childByType(routeNodeTypeAny) != nil {
					goto TryAny
				}

				if sapn != nil {
					goto Struggle
				}
			}

			break
		}

		if s[0] == '/' { // Skip continuous "/"
			for i, sl = 1, len(s); i < sl && s[i] == '/'; i++ {
			}

			s = s[i-1:]
		}

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
			continue
		}

		// Save any parent node for struggling.
		if cn != sapn && cn.childByType(routeNodeTypeAny) != nil {
			sapn = cn
			saps = s
		}

		// Try static node.
		if nn = cn.child(s[0], routeNodeTypeStatic); nn != nil {
			// Save node for struggling.
			if pl = len(cn.prefix); pl > 0 &&
				cn.prefix[pl-1] == '/' {
				sn = cn
				snt = routeNodeTypeParam
				ss = s
			}

			cn = nn

			continue
		}

		// Try param node.
	TryParam:
		if nn = cn.childByType(routeNodeTypeParam); nn != nil {
			// Save node for struggling.
			if pl = len(cn.prefix); pl > 0 &&
				cn.prefix[pl-1] == '/' {
				sn = cn
				snt = routeNodeTypeAny
				ss = s
			}

			cn = nn

			for i, sl = 0, len(s); i < sl && s[i] != '/'; i++ {
			}

			if req.routeParamValues == nil {
				req.routeParamValues = r.allocRouteParamValues()
			}

			if pc < len(cn.paramNames) {
				pc++
			}

			req.routeParamValues = req.routeParamValues[:pc]
			req.routeParamValues[pc-1] = s[:i]

			s = s[i:]

			continue
		}

		// Try any node.
	TryAny:
		if cn = cn.childByType(routeNodeTypeAny); cn != nil {
			if req.routeParamValues == nil {
				req.routeParamValues = r.allocRouteParamValues()
			}

			pc = len(cn.paramNames)
			req.routeParamValues = req.routeParamValues[:pc]
			req.routeParamValues[pc-1] = s

			break
		}

		// Struggle for the former node.
	Struggle:
		if sn != nil {
			cn = sn
			sn = nil
			s = ss
			switch snt {
			case routeNodeTypeParam:
				goto TryParam
			case routeNodeTypeAny:
				goto TryAny
			}
		} else if sapn != nil {
			cn = sapn
			sapn = nil
			s = saps
			goto TryAny
		}

		return r.a.NotFoundHandler
	}

	h := cn.handlers[req.Method]
	if h != nil {
		req.routeParamNames = cn.paramNames
	} else if len(cn.handlers) != 0 {
		h = r.a.MethodNotAllowedHandler
	} else {
		h = r.a.NotFoundHandler
	}

	return h
}

// allocRouteParamValues reuses or creates a string slice for storing route
// param values.
func (r *router) allocRouteParamValues() []string {
	rpvs := r.routeParamValuesPool.Get().([]string)
	if cap(rpvs) < r.maxRouteParams {
		rpvs = r.routeParamValuesPool.New().([]string)
	}

	return rpvs[:0]
}

// routeNode is the node of the route radix tree.
type routeNode struct {
	label      byte
	nType      routeNodeType
	prefix     string
	children   []*routeNode
	paramNames []string
	handlers   map[string]Handler
}

// child returns a child node of the rn by the l and the t.
func (rn *routeNode) child(l byte, t routeNodeType) *routeNode {
	for _, c := range rn.children {
		if c.label == l && c.nType == t {
			return c
		}
	}

	return nil
}

// childByLabel returns a child node of the rn by the l.
func (rn *routeNode) childByLabel(l byte) *routeNode {
	for _, c := range rn.children {
		if c.label == l {
			return c
		}
	}

	return nil
}

// childByType returns a child node of the rn by the t.
func (rn *routeNode) childByType(t routeNodeType) *routeNode {
	for _, c := range rn.children {
		if c.nType == t {
			return c
		}
	}

	return nil
}

// routeNodeType is the type of the `routeNode`.
type routeNodeType uint8

// The route node types.
const (
	routeNodeTypeStatic routeNodeType = iota
	routeNodeTypeParam
	routeNodeTypeAny
)

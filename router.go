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
	routePalPool         *sync.Pool
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
		routePalPool: &sync.Pool{
			New: func() interface{} {
				return &routePal{}
			},
		},
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
	p := r.routePalPool.Get().(*routePal)
	p.s, _ = splitPathQuery(req.Path)
	p.cn = r.routeTree
	p.nnt = routeNodeTypeStatic
	p.sn = nil
	p.ss = ""
	p.pi = 0

	// Search order: static route > param route > any route.
	for {
		if p.s == "" {
			break
		}

		p.i = 1
		p.sl = len(p.s)
		for p.i < p.sl && p.s[p.i] == '/' {
			p.i++
		}

		p.s = p.s[p.i-1:]

		p.pl = 0
		p.ll = 0

		if p.cn.label != ':' {
			p.pl = len(p.cn.prefix)

			p.ml = p.pl
			if p.sl = len(p.s); p.sl < p.ml {
				p.ml = p.sl
			}

			for p.ll < p.ml && p.s[p.ll] == p.cn.prefix[p.ll] {
				p.ll++
			}
		}

		if p.ll != p.pl {
			goto Struggle
		}

		if p.s = p.s[p.ll:]; p.s == "" {
			if len(p.cn.handlers) == 0 {
				if p.cn.childByType(routeNodeTypeParam) != nil {
					goto Param
				}

				if p.cn.childByType(routeNodeTypeAny) != nil {
					goto Any
				}
			}

			break
		}

		// Static node.
		if p.nn = p.cn.child(p.s[0], routeNodeTypeStatic); p.nn != nil {
			// Save next.
			if p.pl = len(p.cn.prefix); p.pl > 0 &&
				p.cn.prefix[p.pl-1] == '/' {
				p.nnt = routeNodeTypeParam
				p.sn = p.cn
				p.ss = p.s
			}

			p.cn = p.nn

			continue
		}

		// Param node.
	Param:
		if p.nn = p.cn.childByType(routeNodeTypeParam); p.nn != nil {
			// Save next.
			if p.pl = len(p.cn.prefix); p.pl > 0 &&
				p.cn.prefix[p.pl-1] == '/' {
				p.nnt = routeNodeTypeAny
				p.sn = p.cn
				p.ss = p.s
			}

			p.cn = p.nn

			p.i = 0
			p.sl = len(p.s)
			for p.i < p.sl && p.s[p.i] != '/' {
				p.i++
			}

			if req.routeParamValues == nil {
				req.routeParamValues = r.routeParamValuesPool.
					Get().([]string)
			}

			req.routeParamValues[p.pi] = p.s[:p.i]
			p.pi++

			p.s = p.s[p.i:]

			continue
		}

		// Any node.
	Any:
		if p.cn = p.cn.childByType(routeNodeTypeAny); p.cn != nil {
			if req.routeParamValues == nil {
				req.routeParamValues = r.routeParamValuesPool.
					Get().([]string)
			}

			req.routeParamValues[len(p.cn.paramNames)-1] = p.s

			break
		}

		// Struggle for the former node.
	Struggle:
		if p.sn != nil {
			p.cn = p.sn
			p.sn = nil
			p.s = p.ss

			switch p.nnt {
			case routeNodeTypeParam:
				goto Param
			case routeNodeTypeAny:
				goto Any
			}
		}

		r.routePalPool.Put(p)

		return r.a.NotFoundHandler
	}

	h := p.cn.handlers[req.Method]
	if h != nil {
		req.routeParamNames = p.cn.paramNames
	} else if len(p.cn.handlers) != 0 {
		h = r.a.MethodNotAllowedHandler
	} else {
		h = r.a.NotFoundHandler
	}

	r.routePalPool.Put(p)

	return h
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

// routePal is the pal struct of the `router#route()`.
type routePal struct {
	s   string        // Search
	cn  *routeNode    // Current node
	nn  *routeNode    // Next node
	nnt routeNodeType // Next node type
	sn  *routeNode    // Saved node
	ss  string        // Saved search
	sl  int           // Search length
	pl  int           // Prefix length
	ll  int           // LCP length
	ml  int           // Minimum length of sl and pl
	i   int           // Index
	pi  int           // Param index
}

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

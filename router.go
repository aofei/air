package air

import (
	"strings"
	"sync"
)

// router is a registry of all registered routes.
type router struct {
	a               *Air
	tree            *node
	routes          map[string]bool
	maxParams       int
	paramValuesPool *sync.Pool
}

// newRouter returns a new instance of the `router` with the a.
func newRouter(a *Air) *router {
	r := &router{
		a: a,
		tree: &node{
			handlers: map[string]Handler{},
		},
		routes: map[string]bool{},
	}
	r.paramValuesPool = &sync.Pool{
		New: func() interface{} {
			return make([]string, 0, r.maxParams)
		},
	}

	return r
}

// register registers a new route for the method and the path with the matching
// h in the r with the optional route-level gases.
func (r *router) register(method, path string, h Handler, gases ...Gas) {
	msg := ""
	if path == "" {
		msg = "air: route path cannot be empty"
	} else if path[0] != '/' {
		msg = "air: route path must start with /"
	} else if strings.Contains(path, "//") {
		msg = "air: route path cannot have //"
	} else if strings.Count(path, ":") > 1 {
		ps := strings.Split(path, "/")
		for _, p := range ps {
			if strings.Count(p, ":") > 1 {
				msg = "air: adjacent params in route path " +
					"must be separated by /"
				break
			}
		}
	} else if strings.Contains(path, "*") {
		if strings.Count(path, "*") > 1 {
			msg = "air: only one * is allowed in route path"
		} else if path[len(path)-1] != '*' {
			msg = "air: * can only appear at end of route path"
		} else if strings.Contains(
			path[strings.LastIndex(path, "/"):],
			":",
		) {
			msg = "air: adjacent param and * in route path must " +
				"be separated by /"
		}
	} else if rn := method + pathWithoutParamNames(path); r.routes[rn] {
		msg = "air: route already exists"
	} else {
		r.routes[rn] = true
	}

	if msg != "" {
		panic(msg)
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
	if l := len(paramNames); l > r.maxParams {
		r.maxParams = l
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
			cn.prefix = s
			cn.kind = nk
			if h != nil {
				cn.handlers[method] = h
			}

			cn.paramNames = paramNames
		} else if ll < pl { // Split node
			nn = &node{
				kind:       cn.kind,
				label:      cn.prefix[ll],
				prefix:     cn.prefix[ll:],
				handlers:   cn.handlers,
				parent:     cn,
				children:   cn.children,
				paramNames: cn.paramNames,
			}

			// Reset parent node.
			cn.kind = nodeKindStatic
			cn.label = cn.prefix[0]
			cn.prefix = cn.prefix[:ll]
			cn.children = nil
			cn.handlers = map[string]Handler{}
			cn.paramNames = nil
			cn.children = append(cn.children, nn)

			if ll == sl { // At parent node
				cn.kind = nk
				if h != nil {
					cn.handlers[method] = h
				}

				cn.paramNames = paramNames
			} else { // Create child node
				nn = &node{
					kind:       nk,
					label:      s[ll],
					prefix:     s[ll:],
					handlers:   map[string]Handler{},
					parent:     cn,
					paramNames: paramNames,
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
				kind:       nk,
				label:      s[0],
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
			if h != nil {
				cn.handlers[method] = h
			}

			if len(cn.paramNames) == 0 {
				cn.paramNames = paramNames
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
	)

	pvs := r.paramValuesPool.Get().([]string)[:0] // Param values
	defer r.paramValuesPool.Put(pvs)

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
			if hasLastSlash(cn.prefix) {
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
			if hasLastSlash(cn.prefix) {
				nk = nodeKindAny
				sn = cn
				ss = s
			}

			cn = nn

			for i, sl = 0, len(s); i < sl && s[i] != '/'; i++ {
			}

			pvs = append(pvs, s[:i])
			s = s[i:]

			continue
		}

		// Any node.
	Any:
		if cn = cn.childByKind(nodeKindAny); cn != nil {
			if len(pvs) < len(cn.paramNames) {
				pvs = append(pvs, s)
			} else {
				pvs[len(cn.paramNames)-1] = s
			}

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

	h := cn.handlers[req.Method]
	if h == nil {
		if len(cn.handlers) != 0 {
			return r.a.MethodNotAllowedHandler
		}

		return r.a.NotFoundHandler
	}

	if len(pvs) == 0 {
		return h
	}

	// NOTE: Slow zone.

	if len(req.params) == 0 {
		req.params = make([]*RequestParam, 0, len(pvs))
		for i, pv := range pvs {
			req.params = append(req.params, &RequestParam{
				Name: cn.paramNames[i],
				Values: []*RequestParamValue{
					{
						i: unescape(pv),
					},
				},
			})
		}

		return h
	}

	req.growParams(len(pvs))
	for i, pv := range pvs {
		pn := cn.paramNames[i]
		pvs := []*RequestParamValue{
			{
				i: unescape(pv),
			},
		}
		if p := req.Param(pn); p != nil {
			p.Values = append(pvs, p.Values...)
		} else {
			req.params = append(req.params, &RequestParam{
				Name:   pn,
				Values: pvs,
			})
		}
	}

	return h
}

// node is the node of the radix tree.
type node struct {
	kind       nodeKind
	label      byte
	prefix     string
	handlers   map[string]Handler
	parent     *node
	children   []*node
	paramNames []string
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

// unescape return a normal string unescaped from the s.
func unescape(s string) string {
	// Count the %, check that they are well-formed.
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

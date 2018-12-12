package air

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	a := New()
	r := a.router

	assert.NotNil(t, r)
	assert.NotNil(t, r.a)
	assert.NotNil(t, r.tree)
	assert.NotNil(t, r.tree.handlers)
	assert.NotNil(t, r.routes)
}

func TestRouterRegister(t *testing.T) {
	a := New()
	r := a.router
	m := http.MethodGet
	h := func(req *Request, res *Response) error {
		return res.WriteString("Foobar")
	}

	// Invalid route paths.

	assert.PanicsWithValue(
		t,
		"air: route path cannot be empty",
		func() {
			r.register(m, "", h)
		},
	)

	assert.PanicsWithValue(
		t,
		"air: route path must start with /",
		func() {
			r.register(m, "foobar", h)
		},
	)

	assert.PanicsWithValue(
		t,
		"air: route path cannot have //",
		func() {
			r.register(m, "//foobar", h)
		},
	)

	assert.PanicsWithValue(
		t,
		"air: adjacent params in route path must be separated by /",
		func() {
			r.register(m, "/:foo:bar", h)
		},
	)

	assert.PanicsWithValue(
		t,
		"air: only one * is allowed in route path",
		func() {
			r.register(m, "/foo*/bar*", h)
		},
	)

	assert.PanicsWithValue(
		t,
		"air: * can only appear at end of route path",
		func() {
			r.register(m, "/foo*/bar", h)
		},
	)

	assert.PanicsWithValue(
		t,
		"air: adjacent param and * in route path must be separated by "+
			"/",
		func() {
			r.register(m, "/:foobar*", h)
		},
	)

	// Duplicate routes.

	r.register(m, "/foobar", h)
	assert.PanicsWithValue(
		t,
		"air: route already exists",
		func() {
			r.register(m, "/foobar", h)
		},
	)

	// Duplicate route param names.

	assert.PanicsWithValue(
		t,
		"air: route path cannot have duplicate param names",
		func() {
			r.register(m, "/:foobar/:foobar", h)
		},
	)

	// Nothing wrong.

	r.register(m, "/:foobar", h)
	r.register(m, "/foo/:bar/*", h)
}

func TestRouterRouteStatic(t *testing.T) {
	a := New()
	r := a.router

	r.register(
		http.MethodGet,
		"/",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /]")
		},
	)

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "//", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.Error(t, r.route(req)(req, res), "Not Found")
	assert.Equal(t, http.StatusNotFound, res.Status)
	assert.Empty(t, rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodHead, "/", nil)
	assert.Error(t, r.route(req)(req, res), "Method Not Allowed")
	assert.Equal(t, http.StatusMethodNotAllowed, res.Status)
	assert.Empty(t, rec.Body.String())
}

func TestRouterRouteParam(t *testing.T) {
	a := New()
	r := a.router

	r.register(
		http.MethodGet,
		"/:foobar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /:foobar]")
		},
	)

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foobar"))
	assert.NotNil(t, req.Param("foobar").Value())
	assert.Empty(t, req.Param("foobar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foobar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "//", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foobar"))
	assert.NotNil(t, req.Param("foobar").Value())
	assert.Empty(t, req.Param("foobar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foobar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foobar"))
	assert.NotNil(t, req.Param("foobar").Value())
	assert.Equal(t, "foobar", req.Param("foobar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foobar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar/", nil)
	assert.Error(t, r.route(req)(req, res), "Not Found")
	assert.Equal(t, http.StatusNotFound, res.Status)
	assert.Empty(t, rec.Body.String())

	r.register(
		http.MethodGet,
		"/foo:bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo:bar]")
		},
	)

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Empty(t, req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo:bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Equal(t, "bar", req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo:bar]", rec.Body.String())
}

func TestRouterRouteAny(t *testing.T) {
	a := New()
	r := a.router

	r.register(
		http.MethodGet,
		"/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /*]")
		},
	)

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Empty(t, req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "//", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Empty(t, req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar/", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar//", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar//", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foo/bar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foo/bar/", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar//", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foo/bar//", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	r.register(
		http.MethodGet,
		"/foobar*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar*]")
		},
	)

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Empty(t, req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "/", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar//", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "//", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar*]", rec.Body.String())

	r.register(
		http.MethodGet,
		"/foobar/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar/*]")
		},
	)

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Empty(t, req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar*]", rec.Body.String())
}

func TestRouterRouteMix(t *testing.T) {
	a := New()
	r := a.router

	r.register(
		http.MethodGet,
		"/",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /]")
		},
	)

	r.register(
		http.MethodGet,
		"/foo",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo]")
		},
	)

	r.register(
		http.MethodGet,
		"/bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /bar]")
		},
	)

	r.register(
		http.MethodGet,
		"/foobar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar]")
		},
	)

	r.register(
		http.MethodGet,
		"/:foobar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /:foobar]")
		},
	)

	r.register(
		http.MethodGet,
		"/foo/:bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo/:bar]")
		},
	)

	r.register(
		http.MethodGet,
		"/foo:bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo:bar]")
		},
	)

	r.register(
		http.MethodGet,
		"/:foo/:bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /:foo/:bar]")
		},
	)

	r.register(
		http.MethodGet,
		"/foobar*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar*]")
		},
	)

	r.register(
		http.MethodGet,
		"/foobar/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar/*]")
		},
	)

	r.register(
		http.MethodGet,
		"/foo/:bar/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo/:bar/*]")
		},
	)

	r.register(
		http.MethodGet,
		"/foo:bar/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo:bar/*]")
		},
	)

	r.register(
		http.MethodGet,
		"/:foo/:bar/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /:foo/:bar/*]")
		},
	)

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/bar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/barfoo", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foobar"))
	assert.NotNil(t, req.Param("foobar").Value())
	assert.Equal(t, "barfoo", req.Param("foobar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foobar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Empty(t, req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo/:bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Equal(t, "bar", req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo/:bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/fooobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Equal(t, "obar", req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo:bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/bar/foo", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foo"))
	assert.NotNil(t, req.Param("foo").Value())
	assert.Equal(t, "bar", req.Param("foo").Value().String())
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Equal(t, "foo", req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foo/:bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobarfoobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar/*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo/:bar/*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foofoobar/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo:bar/*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/bar/foo/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foo"))
	assert.NotNil(t, req.Param("foo").Value())
	assert.Equal(t, "bar", req.Param("foo").Value().String())
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Equal(t, "foo", req.Param("bar").Value().String())
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foo/:bar/*]", rec.Body.String())
}

func TestNodeChild(t *testing.T) {
	n := &node{}
	n.children = append(n.children, &node{
		label: 'a',
		kind:  nodeKindStatic,
	})

	assert.NotNil(t, n.child('a', nodeKindStatic))
	assert.Nil(t, n.child('b', nodeKindParam))

	assert.NotNil(t, n.childByLabel('a'))
	assert.Nil(t, n.childByLabel('b'))

	assert.NotNil(t, n.childByKind(nodeKindStatic))
	assert.Nil(t, n.childByKind(nodeKindParam))
}

func TestHasLastSlash(t *testing.T) {
	assert.True(t, hasLastSlash("/"))
	assert.False(t, hasLastSlash("/foobar"))
}

func TestPathWithoutParamNames(t *testing.T) {
	assert.Equal(t, "/foo/:", pathWithoutParamNames("/foo/:bar"))
}

func TestSplitPathQuery(t *testing.T) {
	p, q := splitPathQuery("/foobar")
	assert.Equal(t, "/foobar", p)
	assert.Empty(t, q)

	p, q = splitPathQuery("/foobar?")
	assert.Equal(t, "/foobar", p)
	assert.Empty(t, q)

	p, q = splitPathQuery("/foobar?foo=bar")
	assert.Equal(t, "/foobar", p)
	assert.Equal(t, "foo=bar", q)
}

func TestUnescape(t *testing.T) {
	assert.Equal(t, "Hello, 世界", unescape("Hello%2C+%E4%B8%96%E7%95%8C"))
}

func fakeRRCycle(
	a *Air,
	method string,
	target string,
	body io.Reader,
) (*Request, *Response, *httptest.ResponseRecorder) {
	req := &Request{
		Air: a,

		parseParamsOnce: &sync.Once{},
	}
	req.SetHTTPRequest(httptest.NewRequest(method, target, body))

	rec := httptest.NewRecorder()
	res := &Response{
		Air:    a,
		Status: http.StatusOK,

		req:  req,
		ohrw: rec,
	}
	res.SetHTTPResponseWriter(&responseWriter{
		r: res,
		w: rec,
	})

	req.res = res

	return req, res, rec
}

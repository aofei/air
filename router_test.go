package air

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	a := New()
	r := a.router

	assert.NotNil(t, r)
	assert.NotNil(t, r.a)
	assert.NotNil(t, r.routeTree)
	assert.NotNil(t, r.routeTree.handlers)
	assert.NotNil(t, r.registeredRoutes)
	assert.NotNil(t, r.routeParamValuesPool)
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
		"air: route handler cannot be nil",
		func() {
			r.register(m, "/foobar", nil)
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
		"air: adjacent param names in route path must be separated by "+
			"/",
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
		"air: adjacent param name and * in route path must be "+
			"separated by /",
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
		"/foobar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar]")
		},
	)

	r.register(
		http.MethodGet,
		"/foo/bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foo/bar]")
		},
	)

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
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foo/bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo", nil)
	assert.Error(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusNotFound, res.Status)
	assert.Empty(t, rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar/foobar", nil)
	assert.Error(t, r.route(req)(req, res))
	assert.Equal(t, http.StatusNotFound, res.Status)
	assert.Empty(t, rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodHead, "/", nil)
	assert.Error(t, r.route(req)(req, res))
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
	assert.Error(t, r.route(req)(req, res))
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

	r.register(
		http.MethodGet,
		"/:foo/:bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /:foo/:bar]")
		},
	)

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("foo"))
	assert.NotNil(t, req.Param("foo").Value())
	assert.NotNil(t, req.Param("bar"))
	assert.NotNil(t, req.Param("bar").Value())
	assert.Equal(t, "foo", req.Param("foo").Value().String())
	assert.Equal(t, "bar", req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foo/:bar]", rec.Body.String())
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

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Empty(t, req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar/*]", rec.Body.String())

	r.register(
		http.MethodGet,
		"/foobar2/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /foobar2/*]")
		},
	)

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar2/", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.NotNil(t, req.Param("*"))
	assert.NotNil(t, req.Param("*").Value())
	assert.Empty(t, req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /foobar2/*]", rec.Body.String())
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
		func(next Handler) Handler {
			return func(req *Request, res *Response) error {
				res.Header.Set("Foo", "bar")
				return next(req, res)
			}
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
	assert.Equal(t, "bar", res.Header.Get("Foo"))
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

func TestRouterRouteFallBackToAny(t *testing.T) {
	a := New()
	r := a.router

	r.register(
		http.MethodGet,
		"/*",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /*]")
		},
	)

	r.register(
		http.MethodGet,
		"/:foo/:bar",
		func(_ *Request, res *Response) error {
			return res.WriteString("Matched [GET /:foo/:bar]")
		},
	)

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/foo", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, "foo", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, "foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, "foo", req.Param("foo").Value().String())
	assert.Equal(t, "bar", req.Param("bar").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /:foo/:bar]", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/foo/bar/foobar", nil)
	assert.NoError(t, r.route(req)(req, res))
	assert.Equal(t, "foo/bar/foobar", req.Param("*").Value().String())
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "Matched [GET /*]", rec.Body.String())
}

func TestRouterAllocRouteParamValues(t *testing.T) {
	a := New()
	r := a.router

	rpvs := r.allocRouteParamValues()
	assert.Len(t, rpvs, 0)
	assert.Zero(t, cap(rpvs))

	r.maxRouteParams++

	rpvs = r.allocRouteParamValues()
	assert.Len(t, rpvs, 0)
	assert.Equal(t, 1, cap(rpvs))

	r.routeParamValuesPool.Put(rpvs)
	r.maxRouteParams++

	rpvs = r.allocRouteParamValues()
	assert.Len(t, rpvs, 0)
	assert.Equal(t, 2, cap(rpvs))
}

func TestRouteNodeChild(t *testing.T) {
	n := &routeNode{}
	n.children = append(n.children, &routeNode{
		label: 'a',
		nType: routeNodeTypeStatic,
	})

	assert.NotNil(t, n.child('a', routeNodeTypeStatic))
	assert.Nil(t, n.child('b', routeNodeTypeParam))

	assert.NotNil(t, n.childByLabel('a'))
	assert.Nil(t, n.childByLabel('b'))

	assert.NotNil(t, n.childByType(routeNodeTypeStatic))
	assert.Nil(t, n.childByType(routeNodeTypeParam))
}

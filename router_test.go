package air

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouterCheckPath(t *testing.T) {
	a := New()
	r := a.router

	assert.Panics(t, func() { r.add("GET", "", nil) })
	assert.Panics(t, func() { r.add("GET", "foobar", nil) })
	assert.Panics(t, func() { r.add("GET", "/foobar/", nil) })
	assert.Panics(t, func() { r.add("GET", "//foobar", nil) })
	assert.Panics(t, func() { r.add("GET", "/:foo:bar", nil) })
	assert.Panics(t, func() { r.add("GET", "/foo*/bar", nil) })
	assert.Panics(t, func() { r.add("GET", "/foo*/bar*", nil) })
	assert.Panics(t, func() { r.add("GET", "/:foobar*", nil) })
	assert.NotPanics(t, func() { r.add("GET", "/:foo", nil) })
	assert.Panics(t, func() { r.add("GET", "/:foo", nil) })
	assert.Panics(t, func() { r.add("GET", "/:bar", nil) })
	assert.Panics(t, func() { r.add("GET", "/:foobar/:foobar", nil) })
}

func TestRouterMatchStatic(t *testing.T) {
	a := New()
	r := a.router

	path := "/foo/bar.jpg"
	r.add("GET", path, func(req *Request, res *Response) error {
		req.Values["path"] = path
		return nil
	})

	req := newRequest(a, httptest.NewRequest("GET", path, nil))
	r.route(req)(req, nil)
	assert.Equal(t, path, req.Values["path"])

	req = newRequest(a, httptest.NewRequest("GET", path, nil))
	r.route(req)(req, nil)
	assert.Equal(t, path, req.Values["path"])
}

func TestRouterMatchParam(t *testing.T) {
	a := New()
	r := a.router

	r.add("GET", "/users/:id", func(*Request, *Response) error {
		return nil
	})

	req := newRequest(a, httptest.NewRequest("GET", "/users/1", nil))
	r.route(req)
	assert.Equal(t, "1", req.PathParams["id"])

	r.add("GET", "/users/search/:keyword", func(*Request, *Response) error {
		return nil
	})

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/search/frameworks/air",
		nil,
	))
	assert.Equal(t, NotFoundHandler(req, nil), r.route(req)(req, nil))

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/search/"+url.PathEscape("Air / Hello+世界"),
		nil,
	))
	r.route(req)
	assert.Equal(t, "Air / Hello 世界", req.PathParams["keyword"])
	assert.Empty(t, req.PathParams["unknown"])

	r.add(
		"GET",
		"/users/:uid/posts/:pid/:anchor",
		func(*Request, *Response) error {
			return nil
		},
	)

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/1/posts/1/stars",
		nil,
	))
	r.route(req)
	assert.Equal(t, "1", req.PathParams["uid"])
	assert.Equal(t, "1", req.PathParams["pid"])
	assert.Equal(t, "stars", req.PathParams["anchor"])
}

func TestRouterMatchAny(t *testing.T) {
	a := New()
	r := a.router

	r.add("GET", "/*", func(*Request, *Response) error {
		return nil
	})

	req := newRequest(a, httptest.NewRequest("GET", "/any", nil))
	r.route(req)
	assert.Equal(t, "any", req.PathParams["*"])

	req = newRequest(a, httptest.NewRequest("GET", "/any//", nil))
	r.route(req)
	assert.Equal(t, "any//", req.PathParams["*"])

	r.add("GET", "/users", func(req *Request, res *Response) error {
		req.Values["kind"] = "static"
		return nil
	})

	r.add("GET", "/users/*", func(req *Request, res *Response) error {
		req.Values["kind"] = "any"
		return nil
	})

	req = newRequest(a, httptest.NewRequest("POST", "/users/", nil))
	assert.Equal(
		t,
		MethodNotAllowedHandler(req, nil),
		r.route(req)(req, nil),
	)

	req = newRequest(a, httptest.NewRequest("GET", "/users/", nil))
	r.route(req)(req, nil)
	assert.Equal(t, "static", req.Values["kind"])

	req = newRequest(a, httptest.NewRequest("GET", "/users/1", nil))
	r.route(req)(req, nil)
	assert.Equal(t, "1", req.PathParams["*"])
	assert.Equal(t, "any", req.Values["kind"])
}

func TestRouterMixMatchParamAndAny(t *testing.T) {
	a := New()
	r := a.router

	r.add(
		"GET",
		"/users/:id/posts/lucky",
		func(req *Request, res *Response) error {
			req.Values["n"] = 1
			return nil
		},
	)

	r.add(
		"GET",
		"/users/:id/posts/:pid",
		func(req *Request, res *Response) error {
			req.Values["n"] = 2
			return nil
		},
	)

	r.add(
		"GET",
		"/users/:id/posts/:pid/comments",
		func(req *Request, res *Response) error {
			req.Values["n"] = 3
			return nil
		},
	)

	r.add(
		"GET",
		"/users/:id/posts/*",
		func(req *Request, res *Response) error {
			req.Values["n"] = 4
			return nil
		},
	)

	req := newRequest(a, httptest.NewRequest(
		"GET",
		"/users/1/posts/lucky",
		nil,
	))
	r.route(req)(req, nil)
	assert.Equal(t, "1", req.PathParams["id"])
	assert.Equal(t, "", req.PathParams["*"])
	assert.Equal(t, 1, req.Values["n"])

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/1/posts/2",
		nil,
	))
	r.route(req)(req, nil)
	assert.Equal(t, "1", req.PathParams["id"])
	assert.Equal(t, "2", req.PathParams["pid"])
	assert.Equal(t, "", req.PathParams["*"])
	assert.Equal(t, 2, req.Values["n"])

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/1/posts/lucky/comments",
		nil,
	))
	r.route(req)(req, nil)
	assert.Equal(t, "1", req.PathParams["id"])
	assert.Equal(t, "lucky", req.PathParams["pid"])
	assert.Equal(t, "", req.PathParams["*"])
	assert.Equal(t, 3, req.Values["n"])

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/1/posts/2/comments/3",
		nil,
	))
	r.route(req)(req, nil)
	assert.Equal(t, "1", req.PathParams["id"])
	assert.Equal(t, "", req.PathParams["pid"])
	assert.Equal(t, "2/comments/3", req.PathParams["*"])
	assert.Equal(t, 4, req.Values["n"])
}

func TestRouterMatchingPriority(t *testing.T) {
	a := New()
	r := a.router

	r.add("GET", "/users", func(req *Request, res *Response) error {
		req.Values["a"] = 1
		return nil
	})

	req := newRequest(a, httptest.NewRequest("GET", "/users", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 1, req.Values["a"])

	r.add("GET", "/users/new", func(req *Request, res *Response) error {
		req.Values["b"] = 2
		return nil
	})

	req = newRequest(a, httptest.NewRequest("GET", "/users/new", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 2, req.Values["b"])

	r.add("GET", "/users/:id", func(req *Request, res *Response) error {
		req.Values["c"] = 3
		return nil
	})

	req = newRequest(a, httptest.NewRequest("GET", "/users/1", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 3, req.Values["c"])

	r.add("GET", "/users/update", func(req *Request, res *Response) error {
		req.Values["d"] = 4
		return nil
	})

	req = newRequest(a, httptest.NewRequest("GET", "/users/update", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 4, req.Values["d"])

	r.add("GET", "/users/delete", func(req *Request, res *Response) error {
		req.Values["e"] = 5
		return nil
	})

	req = newRequest(a, httptest.NewRequest("GET", "/users/del", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 3, req.Values["c"])

	r.add(
		"GET",
		"/users/:id/posts",
		func(req *Request, res *Response) error {
			req.Values["f"] = 6
			return nil
		},
	)

	req = newRequest(a, httptest.NewRequest("GET", "/users/1/posts", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 6, req.Values["f"])

	r.add("GET", "/users/*", func(req *Request, res *Response) error {
		req.Values["g"] = 7
		return nil
	})

	req = newRequest(a, httptest.NewRequest("GET", "/users/1/posts", nil))
	r.route(req)(req, nil)
	assert.Equal(t, 6, req.Values["f"])

	r.add("GET", "/users/*", func(req *Request, res *Response) error {
		req.Values["h"] = 8
		return nil
	})

	req = newRequest(a, httptest.NewRequest(
		"GET",
		"/users/1/followers",
		nil,
	))
	r.route(req)(req, nil)
	assert.Equal(t, 8, req.Values["h"])
	assert.Equal(t, "1/followers", req.PathParams["*"])
}

func TestRouterPathClean(t *testing.T) {
	assert.Equal(t, "/", pathClean(""))
	assert.Equal(t, "/users", pathClean("users"))
}

func TestRouterUnescape(t *testing.T) {
	assert.Empty(t, unescape("%%%%"))
}

func TestRouterIshex(t *testing.T) {
	assert.True(t, ishex('a'))
	assert.False(t, ishex(' '))
}

func TestRouterUnhex(t *testing.T) {
	assert.Equal(t, byte(10), unhex('a'))
	assert.Equal(t, byte(0), unhex(' '))
}

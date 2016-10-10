package air

import "sync"

// Pool represents the pools of `Air`.
type Pool struct {
	contextPool        *sync.Pool
	requestPool        *sync.Pool
	responsePool       *sync.Pool
	requestHeaderPool  *sync.Pool
	responseHeaderPool *sync.Pool
	uriPool            *sync.Pool
	cookiePool         *sync.Pool
}

// newPool returnes a new instance of `Pool`.
func newPool(a *Air) *Pool {
	return &Pool{
		contextPool: &sync.Pool{
			New: func() interface{} {
				return newContext(a)
			},
		},
		requestPool: &sync.Pool{
			New: func() interface{} {
				return newRequest(a)
			},
		},
		responsePool: &sync.Pool{
			New: func() interface{} {
				return newResponse(a)
			},
		},
		requestHeaderPool: &sync.Pool{
			New: func() interface{} {
				return newRequestHeader()
			},
		},
		responseHeaderPool: &sync.Pool{
			New: func() interface{} {
				return newResponseHeader()
			},
		},
		uriPool: &sync.Pool{
			New: func() interface{} {
				return newURI()
			},
		},
		cookiePool: &sync.Pool{
			New: func() interface{} {
				return newCookie()
			},
		},
	}
}

// Context returns an empty instance of `Context` from p.
func (p *Pool) Context() *Context {
	return p.contextPool.Get().(*Context)
}

// Request returns an empty instance of `Request` from p.
func (p *Pool) Request() *Request {
	return p.requestPool.Get().(*Request)
}

// Response returns an empty instance of `Response` from p.
func (p *Pool) Response() *Response {
	return p.responsePool.Get().(*Response)
}

// RequestHeader returns an empty instance of `RequestHeader` from p.
func (p *Pool) RequestHeader() *RequestHeader {
	return p.requestHeaderPool.Get().(*RequestHeader)
}

// ResponseHeader returns an empty instance of `ResponseHeader` from p.
func (p *Pool) ResponseHeader() *ResponseHeader {
	return p.responseHeaderPool.Get().(*ResponseHeader)
}

// URI returns an empty instance of `URI` from p.
func (p *Pool) URI() *URI {
	return p.uriPool.Get().(*URI)
}

// Cookie returns an empty instance of `Cookie` from p.
func (p *Pool) Cookie() *Cookie {
	return p.cookiePool.Get().(*Cookie)
}

// Put puts x back to p.
func (p *Pool) Put(x interface{}) {
	switch v := x.(type) {
	case *Context:
		v.reset()
		p.contextPool.Put(v)
	case *Request:
		v.reset()
		p.requestPool.Put(v)
	case *Response:
		v.reset()
		p.responsePool.Put(v)
	case *RequestHeader:
		v.reset()
		p.requestHeaderPool.Put(v)
	case *ResponseHeader:
		v.reset()
		p.responseHeaderPool.Put(v)
	case *URI:
		v.reset()
		p.uriPool.Put(v)
	case *Cookie:
		v.reset()
		p.cookiePool.Put(v)
	}
}

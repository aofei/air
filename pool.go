package air

import "sync"

// pool represents the pools of `Air`.
type pool struct {
	contextPool        *sync.Pool
	requestPool        *sync.Pool
	responsePool       *sync.Pool
	requestHeaderPool  *sync.Pool
	responseHeaderPool *sync.Pool
	uriPool            *sync.Pool
}

// newPool returnes a new instance of `pool`.
func newPool(a *Air) *pool {
	return &pool{
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
				return &RequestHeader{}
			},
		},
		responseHeaderPool: &sync.Pool{
			New: func() interface{} {
				return &ResponseHeader{}
			},
		},
		uriPool: &sync.Pool{
			New: func() interface{} {
				return &URI{}
			},
		},
	}
}

// context returns an instance of `Context` from p.
func (p *pool) context() *Context {
	return p.contextPool.Get().(*Context)
}

// request returns an instance of `Request` from p.
func (p *pool) request() *Request {
	return p.requestPool.Get().(*Request)
}

// response returns an instance of `Response` from p.
func (p *pool) response() *Response {
	return p.responsePool.Get().(*Response)
}

// requestHeader returns an instance of `RequestHeader` from p.
func (p *pool) requestHeader() *RequestHeader {
	return p.requestHeaderPool.Get().(*RequestHeader)
}

// responseHeader returns an instance of `ResponseHeader` from p.
func (p *pool) responseHeader() *ResponseHeader {
	return p.responseHeaderPool.Get().(*ResponseHeader)
}

// uri returns an instance of `URI` from p.
func (p *pool) uri() *URI {
	return p.uriPool.Get().(*URI)
}

// put puts x back to p.
func (p *pool) put(x interface{}) {
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
	}
}

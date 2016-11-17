package air

import "sync"

// Pool represents the pools of `Air`.
type Pool struct {
	contextPool *sync.Pool
}

// newPool returnes a new instance of `Pool`.
func newPool(a *Air) *Pool {
	return &Pool{
		contextPool: &sync.Pool{
			New: func() interface{} {
				return newContext(a)
			},
		},
	}
}

// Context returns an empty instance of `Context` from p.
func (p *Pool) Context() *Context {
	return p.contextPool.Get().(*Context)
}

// Put puts x back to p.
func (p *Pool) Put(x interface{}) {
	switch v := x.(type) {
	case *Context:
		v.reset()
		p.contextPool.Put(v)
	}
}

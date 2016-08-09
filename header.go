package air

import "github.com/valyala/fasthttp"

type (
	// RequestHeader represents the current HTTP request header.
	RequestHeader struct {
		fastRequestHeader *fasthttp.RequestHeader
	}

	// ResponseHeader represents the current HTTP response header.
	ResponseHeader struct {
		fastResponseHeader *fasthttp.ResponseHeader
	}
)

// Add adds the key, value pair to the header. It appends to any existing values
// associated with key.
func (h *RequestHeader) Add(key, val string) {
	h.fastRequestHeader.Add(key, val)
}

// Del deletes the values associated with key.
func (h *RequestHeader) Del(key string) {
	h.fastRequestHeader.Del(key)
}

// Set sets the header entries associated with key to the single element value.
// It replaces any existing values associated with key.
func (h *RequestHeader) Set(key, val string) {
	h.fastRequestHeader.Set(key, val)
}

// Get gets the first value associated with the given key. If there are
// no values associated with the key, Get returns "".
func (h *RequestHeader) Get(key string) string {
	return string(h.fastRequestHeader.Peek(key))
}

// Keys returns the header keys.
func (h *RequestHeader) Keys() []string {
	keys := make([]string, h.fastRequestHeader.Len())
	i := 0
	h.fastRequestHeader.VisitAll(func(k, v []byte) {
		keys[i] = string(k)
		i++
	})
	return keys
}

// Contains checks if the header is set.
func (h *RequestHeader) Contains(key string) bool {
	return h.fastRequestHeader.Peek(key) != nil
}

// reset resets the instance of `RequestHeader`.
func (h *RequestHeader) reset() {
	h.fastRequestHeader = nil
}

// Add adds the key, value pair to the header. It appends to any existing values
// associated with key.
func (h *ResponseHeader) Add(key, val string) {
	h.fastResponseHeader.Add(key, val)
}

// Del deletes the values associated with key.
func (h *ResponseHeader) Del(key string) {
	h.fastResponseHeader.Del(key)
}

// Get gets the first value associated with the given key. If there are
// no values associated with the key, Get returns "".
func (h *ResponseHeader) Get(key string) string {
	return string(h.fastResponseHeader.Peek(key))
}

// Set sets the header entries associated with key to the single element value.
// It replaces any existing values associated with key.
func (h *ResponseHeader) Set(key, val string) {
	h.fastResponseHeader.Set(key, val)
}

// Keys returns the header keys.
func (h *ResponseHeader) Keys() []string {
	keys := make([]string, h.fastResponseHeader.Len())
	i := 0
	h.fastResponseHeader.VisitAll(func(k, v []byte) {
		keys[i] = string(k)
		i++
	})
	return keys
}

// Contains checks if the header is set.
func (h *ResponseHeader) Contains(key string) bool {
	return h.fastResponseHeader.Peek(key) != nil
}

// reset resets the instance of `ResponseHeader`.
func (h *ResponseHeader) reset() {
	h.fastResponseHeader = nil
}

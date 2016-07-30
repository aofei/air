package air

import "github.com/valyala/fasthttp"

type (
	// Header defines the interface for HTTP header.
	Header interface {
		// Add adds the key, value pair to the header. It appends to any existing values
		// associated with key.
		Add(string, string)

		// Del deletes the values associated with key.
		Del(string)

		// Set sets the header entries associated with key to the single element value.
		// It replaces any existing values associated with key.
		Set(string, string)

		// Get gets the first value associated with the given key. If there are
		// no values associated with the key, Get returns "".
		Get(string) string

		// Keys returns the header keys.
		Keys() []string

		// Contains checks if the header is set.
		Contains(string) bool
	}

	fastRequestHeader struct {
		*fasthttp.RequestHeader
	}

	fastResponseHeader struct {
		*fasthttp.ResponseHeader
	}
)

func (h *fastRequestHeader) Add(key, val string) {
	h.RequestHeader.Add(key, val)
}

func (h *fastRequestHeader) Del(key string) {
	h.RequestHeader.Del(key)
}

func (h *fastRequestHeader) Set(key, val string) {
	h.RequestHeader.Set(key, val)
}

func (h *fastRequestHeader) Get(key string) string {
	return string(h.RequestHeader.Peek(key))
}

func (h *fastRequestHeader) Keys() []string {
	keys := make([]string, h.RequestHeader.Len())
	i := 0
	h.RequestHeader.VisitAll(func(k, v []byte) {
		keys[i] = string(k)
		i++
	})
	return keys
}

func (h *fastRequestHeader) Contains(key string) bool {
	return h.RequestHeader.Peek(key) != nil
}

func (h *fastRequestHeader) reset(hdr *fasthttp.RequestHeader) {
	h.RequestHeader = hdr
}

func (h *fastResponseHeader) Add(key, val string) {
	h.ResponseHeader.Add(key, val)
}

func (h *fastResponseHeader) Del(key string) {
	h.ResponseHeader.Del(key)
}

func (h *fastResponseHeader) Get(key string) string {
	return string(h.ResponseHeader.Peek(key))
}

func (h *fastResponseHeader) Set(key, val string) {
	h.ResponseHeader.Set(key, val)
}

func (h *fastResponseHeader) Keys() []string {
	keys := make([]string, h.ResponseHeader.Len())
	i := 0
	h.ResponseHeader.VisitAll(func(k, v []byte) {
		keys[i] = string(k)
		i++
	})
	return keys
}

func (h *fastResponseHeader) Contains(key string) bool {
	return h.ResponseHeader.Peek(key) != nil
}

func (h *fastResponseHeader) reset(hdr *fasthttp.ResponseHeader) {
	h.ResponseHeader = hdr
}

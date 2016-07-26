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

	// FastRequestHeader holds `fasthttp.RequestHeader`.
	FastRequestHeader struct {
		*fasthttp.RequestHeader
	}

	// FastResponseHeader holds `fasthttp.ResponseHeader`.
	FastResponseHeader struct {
		*fasthttp.ResponseHeader
	}
)

// Add implements `Header#Add` function.
func (h *FastRequestHeader) Add(key, val string) {
	h.RequestHeader.Add(key, val)
}

// Del implements `Header#Del` function.
func (h *FastRequestHeader) Del(key string) {
	h.RequestHeader.Del(key)
}

// Set implements `Header#Set` function.
func (h *FastRequestHeader) Set(key, val string) {
	h.RequestHeader.Set(key, val)
}

// Get implements `Header#Get` function.
func (h *FastRequestHeader) Get(key string) string {
	return string(h.RequestHeader.Peek(key))
}

// Keys implements `Header#Keys` function.
func (h *FastRequestHeader) Keys() []string {
	keys := make([]string, h.RequestHeader.Len())
	i := 0
	h.RequestHeader.VisitAll(func(k, v []byte) {
		keys[i] = string(k)
		i++
	})
	return keys
}

// Contains implements `Header#Contains` function.
func (h *FastRequestHeader) Contains(key string) bool {
	return h.RequestHeader.Peek(key) != nil
}

func (h *FastRequestHeader) reset(hdr *fasthttp.RequestHeader) {
	h.RequestHeader = hdr
}

// Add implements `Header#Add` function.
func (h *FastResponseHeader) Add(key, val string) {
	h.ResponseHeader.Add(key, val)
}

// Del implements `Header#Del` function.
func (h *FastResponseHeader) Del(key string) {
	h.ResponseHeader.Del(key)
}

// Get implements `Header#Get` function.
func (h *FastResponseHeader) Get(key string) string {
	return string(h.ResponseHeader.Peek(key))
}

// Set implements `Header#Set` function.
func (h *FastResponseHeader) Set(key, val string) {
	h.ResponseHeader.Set(key, val)
}

// Keys implements `Header#Keys` function.
func (h *FastResponseHeader) Keys() []string {
	keys := make([]string, h.ResponseHeader.Len())
	i := 0
	h.ResponseHeader.VisitAll(func(k, v []byte) {
		keys[i] = string(k)
		i++
	})
	return keys
}

// Contains implements `Header#Contains` function.
func (h *FastResponseHeader) Contains(key string) bool {
	return h.ResponseHeader.Peek(key) != nil
}

func (h *FastResponseHeader) reset(hdr *fasthttp.ResponseHeader) {
	h.ResponseHeader = hdr
}

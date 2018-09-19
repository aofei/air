package air

import "strings"

// Headers is an HTTP header map.
type Headers map[string][]string

// Get gets the values associated with the key.
//
// The key is case insensitive and will be canonicalized by the
// `strings.ToLower()`. To use non-canonical keys, access the map directly.
func (hs Headers) Get(key string) []string {
	return hs[strings.ToLower(key)]
}

// Set sets the entries associated with the key to the values.
//
// The key is case insensitive and will be canonicalized by the
// `strings.ToLower()`. To use non-canonical keys, access the map directly.
func (hs Headers) Set(key string, values []string) {
	hs[strings.ToLower(key)] = values
}

// Delete deletes the values associated with the key.
//
// The key is case insensitive and will be canonicalized by the
// `strings.ToLower()`. To use non-canonical keys, access the map directly.
func (hs Headers) Delete(key string) {
	delete(hs, strings.ToLower(key))
}

// First tries to return the first value associated with the key. It returns ""
// if there are no values associated with the key.
//
// The key is case insensitive and will be canonicalized by the
// `strings.ToLower()`. To use non-canonical keys, access the map directly.
func (hs Headers) First(key string) string {
	if vs := hs.Get(key); len(vs) > 0 {
		return vs[0]
	}

	return ""
}

// Append appends the value to the entries associated with the key.
//
// The key is case insensitive and will be canonicalized by the
// `strings.ToLower()`. To use non-canonical keys, access the map directly.
func (hs Headers) Append(key string, value string) {
	hs.Set(key, append(hs.Get(key), value))
}

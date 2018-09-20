package air

// Header is an HTTP header.
type Header struct {
	Name   string
	Values []string
}

// FirstValue returns the first value of the h. It returns "" if the h is nil or
// there are no values.
func (h *Header) FirstValue() string {
	if h == nil || len(h.Values) == 0 {
		return ""
	}

	return h.Values[0]
}

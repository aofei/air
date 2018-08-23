package air

import (
	"bytes"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Cookie is an HTTP cookie.
type Cookie struct {
	Name     string
	Value    string
	Expires  time.Time
	MaxAge   int
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
}

// String returns the serialization string of the c.
func (c *Cookie) String() string {
	if !validCookieName(c.Name) {
		return ""
	}

	buf := bytes.Buffer{}

	n := strings.Replace(c.Name, "\r", "-", -1)
	n = strings.Replace(n, "\n", "-", -1)
	v := sanitize(c.Value, func(b byte) bool {
		return validCookieValue(string(b))
	})
	if strings.IndexByte(v, ' ') >= 0 || strings.IndexByte(v, ',') >= 0 {
		v = `"` + v + `"`
	}

	buf.WriteString(n)
	buf.WriteRune('=')
	buf.WriteString(v)

	if len(c.Path) > 0 {
		buf.WriteString("; Path=")
		buf.WriteString(sanitize(c.Path, func(b byte) bool {
			return 0x20 <= b && b < 0x7f && b != ';'
		}))
	}

	if validCookieDomain(c.Domain) {
		d := c.Domain
		if d[0] == '.' {
			d = d[1:]
		}

		buf.WriteString("; Domain=")
		buf.WriteString(d)
	}

	if c.Expires.Year() >= 1601 {
		buf.WriteString("; Expires=")
		buf2 := buf.Bytes()
		buf.Reset()
		buf.Write(c.Expires.UTC().AppendFormat(buf2, http.TimeFormat))
	}

	if c.MaxAge > 0 {
		buf.WriteString("; Max-Age=")
		buf2 := buf.Bytes()
		buf.Reset()
		buf.Write(strconv.AppendInt(buf2, int64(c.MaxAge), 10))
	} else if c.MaxAge < 0 {
		buf.WriteString("; Max-Age=0")
	}

	if c.HTTPOnly {
		buf.WriteString("; HttpOnly")
	}

	if c.Secure {
		buf.WriteString("; Secure")
	}

	return buf.String()
}

// validCookieName returns whether the n is a valid cookie name.
func validCookieName(n string) bool {
	return n != "" && strings.IndexFunc(n, func(r rune) bool {
		return !strings.ContainsRune(
			"!#$%&'*+-."+
				"0123456789"+
				"ABCDEFGHIJKLMNOPQRSTUWVXYZ"+
				"^_`"+
				"abcdefghijklmnopqrstuvwxyz"+
				"|~",
			r,
		)
	}) < 0
}

// validCookieValue returns whether the v is a valid cookie value.
func validCookieValue(v string) bool {
	for _, b := range v {
		if 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\' {
			return true
		}
	}

	return false
}

// validCookieDomain returns whether the d is a valid cookie domain.
func validCookieDomain(d string) bool {
	if l := len(d); l == 0 || l > 255 {
		return false
	}

	if net.ParseIP(d) != nil && !strings.Contains(d, ":") {
		return true
	}

	if d[0] == '.' {
		// A cookie a domain attribute may start with a leading dot.
		d = d[1:]
	}

	ok := false // Ok once we have seen a letter.
	last := byte('.')
	partlen := 0
	for i := 0; i < len(d); i++ {
		c := d[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
			// No '_' allowed here (in contrast to package net).
			ok = true
			partlen++
		case '0' <= c && c <= '9':
			// Fine
			partlen++
		case c == '-':
			// Byte before dash cannot be dot.
			if last == '.' {
				return false
			}

			partlen++
		case c == '.':
			// Byte before dot cannot be dot, dash.
			if last == '.' || last == '-' {
				return false
			}

			if partlen > 63 || partlen == 0 {
				return false
			}

			partlen = 0
		default:
			return false
		}

		last = c
	}

	if last == '-' || partlen > 63 {
		return false
	}

	return ok
}

// sanitize sanitizes the s based on the valid.
func sanitize(s string, valid func(byte) bool) string {
	ok := true
	for i := 0; i < len(s); i++ {
		if !valid(s[i]) {
			ok = false
			break
		}
	}

	if ok {
		return s
	}

	buf := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if b := s[i]; valid(b) {
			buf = append(buf, b)
		}
	}

	return string(buf)
}

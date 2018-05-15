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

// newCookie returns a new instance of the `Cookie`.
func newCookie(raw string) *Cookie {
	parts := strings.Split(strings.TrimSpace(raw), ";")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.TrimSpace(parts[i])
		if len(parts[i]) == 0 {
			continue
		}
		n, v := parts[i], ""
		if i := strings.Index(n, "="); i >= 0 {
			n, v = n[:i], n[i+1:]
		}
		if !validCookieName(n) {
			continue
		}
		if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}
		for i := 0; i < len(v); i++ {
			if !validCookieValueByte(v[i]) {
				continue
			}
		}
		return &Cookie{
			Name:  n,
			Value: v,
		}
	}
	return nil
}

// String returns the serialization string of the c.
func (c *Cookie) String() string {
	if !validCookieName(c.Name) {
		return ""
	}
	buf := bytes.Buffer{}
	buf.WriteString(sanitizeCookieName(c.Name))
	buf.WriteRune('=')
	buf.WriteString(sanitizeCookieValue(c.Value))
	if len(c.Path) > 0 {
		buf.WriteString("; Path=")
		buf.WriteString(sanitizeCookiePath(c.Path))
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
		b2 := buf.Bytes()
		buf.Reset()
		buf.Write(c.Expires.UTC().AppendFormat(b2, http.TimeFormat))
	}
	if c.MaxAge > 0 {
		buf.WriteString("; Max-Age=")
		b2 := buf.Bytes()
		buf.Reset()
		buf.Write(strconv.AppendInt(b2, int64(c.MaxAge), 10))
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

// isTokenTable is a table of the cookie token indicates.
var isTokenTable = [127]bool{
	'!':  true,
	'#':  true,
	'$':  true,
	'%':  true,
	'&':  true,
	'\'': true,
	'*':  true,
	'+':  true,
	'-':  true,
	'.':  true,
	'0':  true,
	'1':  true,
	'2':  true,
	'3':  true,
	'4':  true,
	'5':  true,
	'6':  true,
	'7':  true,
	'8':  true,
	'9':  true,
	'A':  true,
	'B':  true,
	'C':  true,
	'D':  true,
	'E':  true,
	'F':  true,
	'G':  true,
	'H':  true,
	'I':  true,
	'J':  true,
	'K':  true,
	'L':  true,
	'M':  true,
	'N':  true,
	'O':  true,
	'P':  true,
	'Q':  true,
	'R':  true,
	'S':  true,
	'T':  true,
	'U':  true,
	'W':  true,
	'V':  true,
	'X':  true,
	'Y':  true,
	'Z':  true,
	'^':  true,
	'_':  true,
	'`':  true,
	'a':  true,
	'b':  true,
	'c':  true,
	'd':  true,
	'e':  true,
	'f':  true,
	'g':  true,
	'h':  true,
	'i':  true,
	'j':  true,
	'k':  true,
	'l':  true,
	'm':  true,
	'n':  true,
	'o':  true,
	'p':  true,
	'q':  true,
	'r':  true,
	's':  true,
	't':  true,
	'u':  true,
	'v':  true,
	'w':  true,
	'x':  true,
	'y':  true,
	'z':  true,
	'|':  true,
	'~':  true,
}

// validCookieName returns whether the n is a valid cookie name.
func validCookieName(n string) bool {
	return n != "" && strings.IndexFunc(n, func(r rune) bool {
		i := int(r)
		return i >= len(isTokenTable) || !isTokenTable[i]
	}) < 0
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
	ok := false // Ok once we've seen a letter.
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
			// fine
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

// validCookieValueByte returns whether the b is a valid cookie value byte.
func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}

// validCookiePathByte returns whether the b is a valid cookie path byte.
func validCookiePathByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != ';'
}

// sanitizeCookieName sanitizes the n as a cookie name.
func sanitizeCookieName(n string) string {
	return strings.Replace(strings.Replace(n, "\n", "-", -1), "\r", "-", -1)
}

// sanitizeCookieValue sanitizes the v as a cookie value.
func sanitizeCookieValue(v string) string {
	v = sanitizeOrWarn(v, validCookieValueByte)
	if len(v) == 0 {
		return v
	}
	if strings.IndexByte(v, ' ') >= 0 || strings.IndexByte(v, ',') >= 0 {
		return `"` + v + `"`
	}
	return v
}

// sanitizeCookiePath sanitizes the p as a cookie path.
func sanitizeCookiePath(p string) string {
	return sanitizeOrWarn(p, validCookiePathByte)
}

// sanitizeOrWarn sanitizes or warns the v based on the valid.
func sanitizeOrWarn(v string, valid func(byte) bool) string {
	ok := true
	for i := 0; i < len(v); i++ {
		if !valid(v[i]) {
			ok = false
			break
		}
	}
	if ok {
		return v
	}
	buf := make([]byte, 0, len(v))
	for i := 0; i < len(v); i++ {
		if b := v[i]; valid(b) {
			buf = append(buf, b)
		}
	}
	return string(buf)
}

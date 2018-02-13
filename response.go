package air

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Response is an HTTP response.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Cookies    []*Cookie
	Size       int
	Written    bool

	request       *Request
	writer        http.ResponseWriter
	flusher       http.Flusher
	hijacker      http.Hijacker
	closeNotifier http.CloseNotifier
	pusher        http.Pusher
}

// newResponse returns a new instance of the `Response`.
func newResponse(r *Request, writer http.ResponseWriter) *Response {
	flusher, _ := writer.(http.Flusher)
	hijacker, _ := writer.(http.Hijacker)
	closeNotifier, _ := writer.(http.CloseNotifier)
	pusher, _ := writer.(http.Pusher)

	return &Response{
		StatusCode: 200,
		Headers:    map[string]string{},

		request:       r,
		writer:        writer,
		flusher:       flusher,
		hijacker:      hijacker,
		closeNotifier: closeNotifier,
		pusher:        pusher,
	}
}

// write writes the b to the client.
func (r *Response) write(b []byte) error {
	if !r.Written {
		if !checkPreconditions(r.request, r) {
			r.Headers["Content-Length"] = strconv.Itoa(len(b))
			for k, v := range r.Headers {
				r.writer.Header().Set(k, v)
			}
			for _, c := range r.Cookies {
				if v := c.String(); v != "" {
					r.writer.Header().Add("Set-Cookie", v)
				}
			}
		} else if r.StatusCode == 304 {
			delete(r.Headers, "Content-Type")
			delete(r.Headers, "Content-Length")
		} else if r.StatusCode == 412 {
			return &Error{412, "Precondition Failed"}
		}
		r.writer.WriteHeader(r.StatusCode)
		r.Written = true
	}
	if r.request.Method != "HEAD" && r.StatusCode != 304 {
		n, err := r.writer.Write(b)
		if err != nil {
			return err
		}
		r.Size += n
	}
	return nil
}

// NoContent responds to the client with no content.
func (r *Response) NoContent() error {
	return r.write(nil)
}

// Redirect responds to the client with a redirection to the url.
func (r *Response) Redirect(url string) error {
	r.Headers["Location"] = url
	return r.write(nil)
}

// Blob responds to the client with the contentType content b.
func (r *Response) Blob(contentType string, b []byte) error {
	var err error
	if b, err = theMinifier.minify(contentType, b); err != nil {
		return err
	}
	r.Headers["Content-Type"] = contentType
	return r.write(b)
}

// String responds to the client with the "text/plain" content s.
func (r *Response) String(s string) error {
	return r.Blob("text/plain; charset=utf-8", []byte(s))
}

// JSON responds to the client with the "application/json" content v.
func (r *Response) JSON(v interface{}) error {
	b, err := json.Marshal(v)
	if DebugMode {
		b, err = json.MarshalIndent(v, "", "\t")
	}
	if err != nil {
		return err
	}
	return r.Blob("application/json; charset=utf-8", b)
}

// XML responds to the client with the "application/xml" content v.
func (r *Response) XML(v interface{}) error {
	b, err := xml.Marshal(v)
	if DebugMode {
		b, err = xml.MarshalIndent(v, "", "\t")
	}
	if err != nil {
		return err
	}
	b = append([]byte(xml.Header), b...)
	return r.Blob("application/xml; charset=utf-8", b)
}

// HTML responds to the client with the "text/html" content html.
func (r *Response) HTML(html string) error {
	return r.Blob("text/html; charset=utf-8", []byte(html))
}

// Render renders one or more HTML templates with the m and responds to the
// client with the "text/html" content. The results rendered by the former can
// be inherited by accessing the `m["InheritedHTML"]`.
func (r *Response) Render(m map[string]interface{}, templates ...string) error {
	buf := &bytes.Buffer{}
	for _, t := range templates {
		m["InheritedHTML"] = template.HTML(buf.String())
		buf.Reset()
		if err := theRenderer.render(buf, t, m); err != nil {
			return err
		}
	}
	return r.HTML(buf.String())
}

// Stream responds to the client with the contentType streaming content reader.
func (r *Response) Stream(contentType string, reader io.Reader) error {
	if err := r.Blob(contentType, nil); err != nil {
		return err
	} else if r.request.Method != "HEAD" && r.StatusCode != 304 {
		n, err := io.Copy(r.writer, reader)
		if err != nil {
			return err
		}
		r.Size += int(n)
	}
	return nil
}

// File responds to the client with a file content with the file.
func (r *Response) File(file string) error {
	file, err := filepath.Abs(file)
	if err != nil {
		return err
	} else if fi, err := os.Stat(file); err != nil {
		return err
	} else if fi.IsDir() {
		if p := r.request.request.URL.Path; p[len(p)-1] != '/' {
			p = path.Base(p) + "/"
			if q := r.request.request.URL.RawQuery; q != "" {
				p += "?" + q
			}
			r.StatusCode = 301
			return r.Redirect(p)
		}
		file += "index.html"
	}

	c := []byte{}
	mt := time.Time{}
	if a, err := theCoffer.asset(file); err != nil {
		return err
	} else if a != nil {
		c = a.content
		mt = a.modTime
	} else if fi, err := os.Stat(file); err != nil {
		return err
	} else if c, err = ioutil.ReadFile(file); err != nil {
		return err
	} else {
		mt = fi.ModTime()
	}

	if _, ok := r.Headers["ETag"]; !ok {
		r.Headers["ETag"] = fmt.Sprintf(`"%x"`, md5.Sum(c))
	}

	for k, v := range r.Headers {
		r.writer.Header().Set(k, v)
	}

	for _, c := range r.Cookies {
		if v := c.String(); v != "" {
			r.writer.Header().Add("Set-Cookie", v)
		}
	}

	http.ServeContent(
		r.writer,
		r.request.request,
		file,
		mt,
		bytes.NewReader(c),
	)

	r.Written = true
	if r.request.Method != "HEAD" && r.StatusCode != 304 {
		r.Size += len(c)
	}

	return nil
}

// Flush flushes buffered data to the client.
func (r *Response) Flush() {
	r.flusher.Flush()
}

// Hijack took over the connection from the server.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.hijacker.Hijack()
}

// CloseNotify returns a channel that receives at most a single value when the
// connection has gone away.
func (r *Response) CloseNotify() <-chan bool {
	return r.closeNotifier.CloseNotify()
}

// Push initiates an HTTP/2 server push. This constructs a synthetic request
// using the given target and pos, serializes that request into a PUSH_PROMISE
// frame, then dispatches that request using the server's request handler. If
// pos is nil, default options are used.
func (r *Response) Push(target string, pos *http.PushOptions) error {
	return r.pusher.Push(target, pos)
}

// checkPreconditions evaluates request preconditions and reports whether a
// precondition resulted in sending not modified or precondition failed.
func checkPreconditions(req *Request, res *Response) bool {
	im := req.Headers["If-Match"]
	ius, _ := http.ParseTime(req.Headers["If-Unmodified-Since"])
	if im != "" {
		if !checkIfMatch(res, im) {
			res.StatusCode = 412
			return true
		}
	} else if !ius.IsZero() && !checkIfModifiedSince(res, ius) {
		res.StatusCode = 412
		return true
	}
	inm := req.Headers["If-None-Match"]
	ims, _ := http.ParseTime(req.Headers["If-Modified-Since"])
	if inm != "" {
		if !checkIfNoneMatch(res, inm) {
			if req.Method == "GET" || req.Method == "HEAD" {
				res.StatusCode = 304
				return true
			}
			res.StatusCode = 412
			return true
		}
	} else if !ims.IsZero() &&
		(req.Method == "GET" || req.Method == "HEAD") &&
		!checkIfModifiedSince(res, ims) {
		res.StatusCode = 304
		return true
	}
	return false
}

// checkIfMatch reports whether the im and the ETag in the res match.
func checkIfMatch(res *Response, im string) bool {
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return true
		}
		eTag, remain := scanETag(im)
		if eTag == "" {
			break
		}
		if eTagStrongMatch(eTag, res.Headers["ETag"]) {
			return true
		}
		im = remain
	}
	return false
}

// checkIfUnmodifiedSince reports whether the ius before the Last-Modified in
// the res.
func checkIfUnmodifiedSince(res *Response, ius time.Time) bool {
	lm, _ := http.ParseTime(res.Headers["Last-Modified"])
	return lm.Before(ius.Add(time.Second))
}

// checkIfNoneMatch reports whether the im and the ETag in the res not match.
func checkIfNoneMatch(res *Response, inm string) bool {
	for {
		inm = textproto.TrimString(inm)
		if len(inm) == 0 {
			break
		}
		if inm[0] == ',' {
			inm = inm[1:]
		}
		if inm[0] == '*' {
			return false
		}
		eTag, remain := scanETag(inm)
		if eTag == "" {
			break
		}
		if eTagWeakMatch(eTag, res.Headers["ETag"]) {
			return false
		}
		inm = remain
	}
	return true
}

// checkIfModifiedSince reports whether the ius not before the Last-Modified in
// the res.
func checkIfModifiedSince(res *Response, ims time.Time) bool {
	lm, _ := http.ParseTime(res.Headers["Last-Modified"])
	return !lm.Before(ims.Add(time.Second))
}

// scanETag determines if a syntactically valid ETag is present at s. If so, the
// ETag and remaining text after consuming ETag is returned. Otherwise, it
// returns "", "".
func scanETag(s string) (eTag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	// ETag is either W/"text" or "text".
	// See RFC 7232 2.3.
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		// Character values allowed in ETags.
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return string(s[:i+1]), s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

// eTagStrongMatch reports whether a and b match using strong ETag comparison.
func eTagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

// eTagWeakMatch reports whether a and b match using weak ETag comparison.
func eTagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

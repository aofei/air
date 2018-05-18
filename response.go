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
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Response is an HTTP response.
type Response struct {
	StatusCode    int
	Headers       map[string]string
	ContentLength int64
	Cookies       []*Cookie
	Written       bool

	request            *Request
	httpResponseWriter http.ResponseWriter
}

// Write responds to the client with the content.
func (r *Response) Write(content io.ReadSeeker) error {
	if r.Written {
		return nil
	}

	canWrite := false
	var reader io.Reader = content
	defer func() {
		if !canWrite {
			return
		}

		if reader != nil && r.ContentLength == 0 {
			r.ContentLength, _ = content.Seek(0, io.SeekEnd)
			content.Seek(0, io.SeekStart)
		}

		if r.StatusCode < 400 {
			r.Headers["Accept-Ranges"] = "bytes"
			if r.Headers["Content-Encoding"] == "" {
				r.Headers["Content-Length"] = strconv.FormatInt(
					r.ContentLength,
					10,
				)
			}
		}

		cls := make([]string, 0, len(r.Cookies))
		for _, c := range r.Cookies {
			if v := c.String(); v != "" {
				cls = append(cls, v)
			}
		}

		if len(cls) > 0 {
			r.Headers["Set-Cookie"] = strings.Join(cls, ", ")
		}

		for k, v := range r.Headers {
			if _, ok := r.httpResponseWriter.Header()[k]; !ok {
				r.httpResponseWriter.Header().Set(k, v)
			}
		}

		r.httpResponseWriter.WriteHeader(r.StatusCode)
		if r.request.Method != "HEAD" {
			io.CopyN(r.httpResponseWriter, reader, r.ContentLength)
		}

		r.Written = true
	}()

	if r.StatusCode >= 400 { // something has gone wrong
		canWrite = true
		return nil
	}

	im := r.request.Headers["If-Match"]
	et := r.Headers["ETag"]
	ius, _ := http.ParseTime(r.request.Headers["If-Unmodified-Since"])
	lm, _ := http.ParseTime(r.Headers["Last-Modified"])
	if im != "" {
		matched := false
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
				matched = true
				break
			}
			eTag, remain := scanETag(im)
			if eTag == "" {
				break
			}
			if eTagStrongMatch(eTag, et) {
				matched = true
				break
			}
			im = remain
		}
		if !matched {
			r.StatusCode = 412
		}
	} else if !ius.IsZero() && !lm.Before(ius.Add(time.Second)) {
		r.StatusCode = 412
	}

	inm := r.request.Headers["If-None-Match"]
	ims, _ := http.ParseTime(r.request.Headers["If-Modified-Since"])
	if inm != "" {
		noneMatched := true
		for {
			inm = textproto.TrimString(inm)
			if len(inm) == 0 {
				break
			}
			if inm[0] == ',' {
				inm = inm[1:]
			}
			if inm[0] == '*' {
				noneMatched = false
				break
			}
			eTag, remain := scanETag(inm)
			if eTag == "" {
				break
			}
			if eTagWeakMatch(eTag, r.Headers["ETag"]) {
				noneMatched = false
				break
			}
			inm = remain
		}
		if !noneMatched {
			if m := r.request.Method; m == "GET" || m == "HEAD" {
				r.StatusCode = 304
			} else {
				r.StatusCode = 412
			}
		}
	} else if !ims.IsZero() && lm.Before(ims.Add(time.Second)) {
		r.StatusCode = 304
	}

	if r.StatusCode == 304 {
		delete(r.Headers, "Content-Type")
		delete(r.Headers, "Content-Length")
		canWrite = true
		return nil
	} else if r.StatusCode == 412 {
		return &Error{412, "Precondition Failed"}
	} else if content == nil {
		canWrite = true
		return nil
	}

	ct, ok := r.Headers["Content-Type"]
	if !ok {
		// Read a chunk to decide between UTF-8 text and binary
		b := [1 << 9]byte{}
		n, _ := io.ReadFull(content, b[:])
		ct = http.DetectContentType(b[:n])
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}
		r.Headers["Content-Type"] = ct
	}

	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	} else if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	r.ContentLength = size

	rh := r.request.Headers["Range"]
	if rh == "" {
		canWrite = true
		return nil
	} else if r.request.Method == "GET" || r.request.Method == "HEAD" {
		if ir := r.request.Headers["If-Range"]; ir != "" {
			if eTag, _ := scanETag(ir); eTag != "" &&
				!eTagStrongMatch(eTag, et) {
				canWrite = true
				return nil
			}

			// The If-Range value is typically the ETag value, but
			// it may also be the modtime date. See
			// golang.org/issue/8367.
			if lm.IsZero() {
				canWrite = true
				return nil
			} else if t, _ := http.ParseTime(ir); !t.Equal(lm) {
				canWrite = true
				return nil
			}
		}
	}

	const b = "bytes="
	if !strings.HasPrefix(rh, b) {
		return &Error{416, "Invalid Range"}
	}

	ranges := []httpRange{}
	noOverlap := false
	for _, ra := range strings.Split(rh[len(b):], ",") {
		ra = strings.TrimSpace(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return &Error{416, "Invalid Range"}
		}
		start := strings.TrimSpace(ra[:i])
		end := strings.TrimSpace(ra[i+1:])
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return &Error{416, "Invalid Range"}
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return &Error{416, "Invalid Range"}
			}
			if i >= size {
				// If the range begins after the size of the
				// content, then it does not overlap.
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end
				// of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return &Error{416, "Invalid Range"}
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}

	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		r.Headers["Content-Range"] = fmt.Sprintf("bytes */%d", size)
		return &Error{416, "Invalid Range: Failed to Overlap"}
	}

	var rangesSize int64
	for _, ra := range ranges {
		rangesSize += ra.length
	}

	if rangesSize > size {
		ranges = nil
	}

	if l := len(ranges); l == 1 {
		// RFC 2616, Section 14.16:
		// "When an HTTP message includes the content of a single range
		// (for example, a response to a request for a single range, or
		// to a request for a set of ranges that overlap without any
		// holes), this content is transmitted with a Content-Range
		// header, and a Content-Length header showing the number of
		// bytes actually transferred.
		// ...
		// A response to a request for a single range MUST NOT be sent
		// using the multipart/byteranges media type."
		ra := ranges[0]
		if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
			return &Error{416, err.Error()}
		}
		r.ContentLength = ra.length
		r.StatusCode = 206
		r.Headers["Content-Range"] = ra.contentRange(size)
	} else if l > 1 {
		var w countingWriter
		mw := multipart.NewWriter(&w)
		for _, ra := range ranges {
			mw.CreatePart(ra.header(ct, size))
			r.ContentLength += ra.length
		}
		mw.Close()
		r.ContentLength += int64(w)

		r.StatusCode = 206

		pr, pw := io.Pipe()
		mw = multipart.NewWriter(pw)
		r.Headers["Content-Type"] = "multipart/byteranges; boundary=" +
			mw.Boundary()
		reader = pr
		defer pr.Close()
		go func() {
			for _, ra := range ranges {
				part, err := mw.CreatePart(ra.header(ct, size))
				if err != nil {
					pw.CloseWithError(err)
					return
				}
				if _, err := content.Seek(
					ra.start,
					io.SeekStart,
				); err != nil {
					pw.CloseWithError(err)
					return
				}
				if _, err := io.CopyN(
					part,
					content,
					ra.length,
				); err != nil {
					pw.CloseWithError(err)
					return
				}
			}
			mw.Close()
			pw.Close()
		}()
	}

	canWrite = true
	return nil
}

// NoContent responds to the client with no content.
func (r *Response) NoContent() error {
	return r.Write(nil)
}

// Redirect responds to the client with a redirection to the url.
func (r *Response) Redirect(url string) error {
	r.Headers["Location"] = url
	return r.Write(nil)
}

// Blob responds to the client with the content b.
func (r *Response) Blob(b []byte) error {
	if ct, ok := r.Headers["Content-Type"]; ok {
		var err error
		if b, err = theMinifier.minify(ct, b); err != nil {
			return err
		}
	}
	return r.Write(bytes.NewReader(b))
}

// String responds to the client with the "text/plain" content s.
func (r *Response) String(s string) error {
	r.Headers["Content-Type"] = "text/plain; charset=utf-8"
	return r.Blob([]byte(s))
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
	r.Headers["Content-Type"] = "application/json; charset=utf-8"
	return r.Blob(b)
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
	r.Headers["Content-Type"] = "application/xml; charset=utf-8"
	return r.Blob(b)
}

// HTML responds to the client with the "text/html" content h.
func (r *Response) HTML(h string) error {
	if AutoPushEnabled && r.request.Proto == "HTTP/2" {
		tree, err := html.Parse(strings.NewReader(h))
		if err != nil {
			return err
		}
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				target := ""
				switch n.Data {
				case "link":
					for _, a := range n.Attr {
						if a.Key == "href" {
							target = a.Val
							break
						}
					}
				case "img", "script":
					for _, a := range n.Attr {
						if a.Key == "src" {
							target = a.Val
							break
						}
					}
				}
				if path.IsAbs(target) {
					r.Push(target, nil)
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(tree)
	}
	r.Headers["Content-Type"] = "text/html; charset=utf-8"
	return r.Blob([]byte(h))
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

// File responds to the client with a file content with the file.
func (r *Response) File(file string) error {
	file, err := filepath.Abs(file)
	if err != nil {
		return err
	} else if fi, err := os.Stat(file); err != nil {
		return err
	} else if fi.IsDir() {
		if p := r.request.URL.Path; p[len(p)-1] != '/' {
			p = path.Base(p) + "/"
			if q := r.request.URL.Query; q != "" {
				p += "?" + q
			}
			r.StatusCode = 301
			return r.Redirect(p)
		}
		file += "index.html"
	}

	var c io.ReadSeeker
	mt := time.Time{}
	if a, err := theCoffer.asset(file); err != nil {
		return err
	} else if a != nil {
		c = bytes.NewReader(a.content)
		mt = a.modTime
	} else {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return err
		}

		c = f
		mt = fi.ModTime()
	}

	if ct, ok := r.Headers["Content-Type"]; !ok {
		if ct = mime.TypeByExtension(filepath.Ext(file)); ct == "" {
			// Read a chunk to decide between UTF-8 text and binary
			b := [1 << 9]byte{}
			n, _ := io.ReadFull(c, b[:])
			ct = http.DetectContentType(b[:n])
			if _, err := c.Seek(0, io.SeekStart); err != nil {
				return err
			}
		}
		r.Headers["Content-Type"] = ct
	}

	if _, ok := r.Headers["ETag"]; !ok {
		h := md5.New()
		if _, err := io.Copy(h, c); err != nil {
			return err
		}
		r.Headers["ETag"] = fmt.Sprintf(`"%x"`, h.Sum(nil))
	}

	if _, ok := r.Headers["Last-Modified"]; !ok {
		r.Headers["Last-Modified"] = mt.UTC().Format(http.TimeFormat)
	}

	return r.Write(c)
}

// Flush flushes buffered data to the client.
func (r *Response) Flush() {
	r.httpResponseWriter.(http.Flusher).Flush()
}

// Hijack took over the connection from the server.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.httpResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify returns a channel that receives at most a single value when the
// connection has gone away.
func (r *Response) CloseNotify() <-chan bool {
	return r.httpResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Push initiates an HTTP/2 server push. This constructs a synthetic request
// using the given target and headers, serializes that request into a
// PUSH_PROMISE frame, then dispatches that request using the server's request
// handler. The target must either be an absolute path (like "/path") or an
// absolute URL that contains a valid host and the same scheme as the parent
// request. If the target is a path, it will inherit the scheme and host of the
// parent request. The headers specifies additional promised request headers.
// The headers cannot include HTTP/2 pseudo header fields like ":path" and
// ":scheme", which will be added automatically.
func (r *Response) Push(target string, headers map[string]string) error {
	var pos *http.PushOptions
	for k, v := range headers {
		if pos == nil {
			pos = &http.PushOptions{
				Header: http.Header{},
			}
		}
		pos.Header.Set(k, v)
	}
	return r.httpResponseWriter.(http.Pusher).Push(target, pos)
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

// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

// contentRange return a Content-Range header field of the r.
func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

// header return  the MIME header of the r.
func (r httpRange) header(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		"Content-Type":  {contentType},
	}
}

// countingWriter counts how many bytes have been written to it.
type countingWriter int64

// Write implements the `io.Writer`.
func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

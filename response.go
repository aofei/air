package air

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"errors"
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

	"github.com/gorilla/websocket"
	"golang.org/x/net/html"
)

// Response is an HTTP response.
type Response struct {
	Status        int
	Headers       map[string][]string
	ContentLength int64
	Cookies       map[string]*Cookie
	Written       bool

	request *Request
	writer  http.ResponseWriter
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

		if r.Status >= 200 && r.Status < 400 {
			r.Headers["accept-ranges"] = []string{"bytes"}
		}

		if len(r.Headers["content-encoding"]) == 0 &&
			r.Status >= 200 &&
			r.Status != 204 &&
			(r.Status >= 300 || r.request.Method != "CONNECT") {
			r.Headers["content-length"] = []string{
				strconv.FormatInt(r.ContentLength, 10),
			}
		}

		if len(r.Cookies) > 0 {
			vs := make([]string, 0, len(r.Cookies))
			for _, c := range r.Cookies {
				vs = append(vs, c.String())
			}

			r.Headers["set-cookie"] = vs
		}

		for k, v := range r.Headers {
			for _, v := range v {
				r.writer.Header().Add(k, v)
			}
		}

		r.writer.WriteHeader(r.Status)
		if r.request.Method != "HEAD" {
			io.CopyN(r.writer, reader, r.ContentLength)
		}

		r.Written = true
	}()

	if r.Status >= 400 { // Something has gone wrong
		canWrite = true
		return nil
	}

	im := ""
	if ims := r.request.Headers["if-match"]; len(ims) > 0 {
		im = ims[0]
	}

	et := ""
	if ets := r.Headers["etag"]; len(ets) > 0 {
		et = ets[0]
	}

	ius := time.Time{}
	if iuss := r.request.Headers["if-unmodified-since"]; len(iuss) > 0 {
		ius, _ = http.ParseTime(iuss[0])
	}

	lm := time.Time{}
	if lms := r.Headers["last-modified"]; len(lms) > 0 {
		lm, _ = http.ParseTime(lms[0])
	}

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
			r.Status = 412
		}
	} else if !ius.IsZero() && !lm.Before(ius.Add(time.Second)) {
		r.Status = 412
	}

	inm := ""
	if inms := r.request.Headers["if-none-match"]; len(inms) > 0 {
		inm = inms[0]
	}

	ims := time.Time{}
	if imss := r.request.Headers["if-modified-since"]; len(imss) > 0 {
		ims, _ = http.ParseTime(imss[0])
	}

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

			et := ""
			if ets := r.Headers["etag"]; len(ets) > 0 {
				et = ets[0]
			}

			if eTagWeakMatch(eTag, et) {
				noneMatched = false
				break
			}

			inm = remain
		}
		if !noneMatched {
			if r.request.Method == "GET" ||
				r.request.Method == "HEAD" {
				r.Status = 304
			} else {
				r.Status = 412
			}
		}
	} else if !ims.IsZero() && lm.Before(ims.Add(time.Second)) {
		r.Status = 304
	}

	if r.Status == 304 {
		delete(r.Headers, "content-type")
		delete(r.Headers, "content-length")
		canWrite = true
		return nil
	} else if r.Status == 412 {
		return errors.New("precondition failed")
	} else if content == nil {
		canWrite = true
		return nil
	}

	ct := ""
	if cts := r.Headers["content-type"]; len(cts) > 0 {
		ct = cts[0]
	} else {
		// Read a chunk to decide between UTF-8 text and binary
		b := [1 << 9]byte{}
		n, _ := io.ReadFull(content, b[:])
		ct = http.DetectContentType(b[:n])
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		r.Headers["content-type"] = []string{ct}
	}

	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	} else if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	r.ContentLength = size

	rh := ""
	if rhs := r.request.Headers["range"]; len(rhs) == 0 {
		canWrite = true
		return nil
	} else if r.request.Method == "GET" || r.request.Method == "HEAD" {
		if irs := r.request.Headers["if-range"]; len(irs) > 0 {
			ir := irs[0]
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
	} else {
		rh = rhs[0]
	}

	const b = "bytes="
	if !strings.HasPrefix(rh, b) {
		r.Status = 416
		return errors.New("invalid range")
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
			r.Status = 416
			return errors.New("invalid range")
		}

		start := strings.TrimSpace(ra[:i])
		end := strings.TrimSpace(ra[i+1:])
		hr := httpRange{}
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				r.Status = 416
				return errors.New("invalid range")
			}

			if i > size {
				i = size
			}

			hr.start = size - i
			hr.length = size - hr.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				r.Status = 416
				return errors.New("invalid range")
			}

			if i >= size {
				// If the range begins after the size of the
				// content, then it does not overlap.
				noOverlap = true
				continue
			}

			hr.start = i
			if end == "" {
				// If no end is specified, range extends to end
				// of the file.
				hr.length = size - hr.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || hr.start > i {
					r.Status = 416
					return errors.New("invalid range")
				}

				if i >= size {
					i = size - 1
				}

				hr.length = i - hr.start + 1
			}
		}

		ranges = append(ranges, hr)
	}

	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		r.Headers["content-range"] = []string{
			fmt.Sprintf("bytes */%d", size),
		}

		r.Status = 416

		return errors.New("invalid range: failed to overlap")
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
			r.Status = 416
			return err
		}

		r.ContentLength = ra.length
		r.Status = 206
		r.Headers["content-range"] = []string{ra.contentRange(size)}
	} else if l > 1 {
		var w countingWriter
		mw := multipart.NewWriter(&w)
		for _, ra := range ranges {
			mw.CreatePart(ra.header(ct, size))
			r.ContentLength += ra.length
		}

		mw.Close()
		r.ContentLength += int64(w)

		r.Status = 206

		pr, pw := io.Pipe()
		mw = multipart.NewWriter(pw)
		r.Headers["content-type"] = []string{
			"multipart/byteranges; boundary=" + mw.Boundary(),
		}
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
	r.Headers["location"] = []string{url}
	return r.Write(nil)
}

// Blob responds to the client with the content b.
func (r *Response) Blob(b []byte) error {
	if cts := r.Headers["content-type"]; len(cts) > 0 {
		var err error
		if b, err = theMinifier.minify(cts[0], b); err != nil {
			return err
		}
	}

	return r.Write(bytes.NewReader(b))
}

// String responds to the client with the "text/plain" content s.
func (r *Response) String(s string) error {
	r.Headers["content-type"] = []string{"text/plain; charset=utf-8"}
	return r.Blob([]byte(s))
}

// JSON responds to the client with the "application/json" content v.
func (r *Response) JSON(v interface{}) error {
	var (
		b   []byte
		err error
	)

	if DebugMode {
		b, err = json.MarshalIndent(v, "", "\t")
	} else {
		b, err = json.Marshal(v)
	}

	if err != nil {
		return err
	}

	r.Headers["content-type"] = []string{"application/json; charset=utf-8"}

	return r.Blob(b)
}

// XML responds to the client with the "application/xml" content v.
func (r *Response) XML(v interface{}) error {
	var (
		b   []byte
		err error
	)

	if DebugMode {
		b, err = xml.MarshalIndent(v, "", "\t")
	} else {
		b, err = xml.Marshal(v)
	}

	if err != nil {
		return err
	}

	b = append([]byte(xml.Header), b...)
	r.Headers["content-type"] = []string{"application/xml; charset=utf-8"}

	return r.Blob(b)
}

// HTML responds to the client with the "text/html" content h.
func (r *Response) HTML(h string) error {
	if AutoPushEnabled && r.request.httpRequest.TLS != nil {
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

	r.Headers["content-type"] = []string{"text/html; charset=utf-8"}

	return r.Blob([]byte(h))
}

// Render renders one or more HTML templates with the m and responds to the
// client with the "text/html" content. The results rendered by the former can
// be inherited by accessing the `m["InheritedHTML"]`.
func (r *Response) Render(m map[string]interface{}, templates ...string) error {
	buf := bytes.Buffer{}
	for _, t := range templates {
		m["InheritedHTML"] = template.HTML(buf.String())
		buf.Reset()
		err := theRenderer.render(&buf, t, m, r.request.localizedString)
		if err != nil {
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
		p := r.request.httpRequest.URL.EscapedPath()
		if p[len(p)-1] != '/' {
			p = path.Base(p) + "/"
			if q := r.request.httpRequest.URL.RawQuery; q != "" {
				p += "?" + q
			}

			r.Status = 301

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

	if len(r.Headers["content-type"]) == 0 {
		ct := mime.TypeByExtension(filepath.Ext(file))
		if ct == "" {
			// Read a chunk to decide between UTF-8 text and binary
			b := [1 << 9]byte{}
			n, _ := io.ReadFull(c, b[:])
			ct = http.DetectContentType(b[:n])
			if _, err := c.Seek(0, io.SeekStart); err != nil {
				return err
			}
		}

		r.Headers["content-type"] = []string{ct}
	}

	if len(r.Headers["etag"]) == 0 {
		h := md5.New()
		if _, err := io.Copy(h, c); err != nil {
			return err
		}

		r.Headers["etag"] = []string{fmt.Sprintf(`"%x"`, h.Sum(nil))}
	}

	if len(r.Headers["last-modified"]) == 0 {
		r.Headers["last-modified"] = []string{
			mt.UTC().Format(http.TimeFormat),
		}
	}

	return r.Write(c)
}

// WebSocket tries to convert the connection to the WebSocket protocol.
func (r *Response) WebSocket() (*WebSocketConn, error) {
	r.Status = 101

	if len(r.Cookies) > 0 {
		vs := make([]string, 0, len(r.Cookies))
		for _, c := range r.Cookies {
			vs = append(vs, c.String())
		}

		r.Headers["set-cookie"] = vs
	}

	for k, v := range r.Headers {
		for _, v := range v {
			r.writer.Header().Add(k, v)
		}
	}

	r.Written = true

	conn, err := (&websocket.Upgrader{}).Upgrade(
		r.writer,
		r.request.httpRequest,
		r.writer.Header(),
	)
	if err != nil {
		return nil, err
	}

	wsc := &WebSocketConn{
		conn: conn,
	}

	conn.SetCloseHandler(func(statusCode int, reason string) error {
		if wsc.ConnectionCloseHandler != nil {
			return wsc.ConnectionCloseHandler(statusCode, reason)
		}

		mt := int(WebSocketMessageTypeConnectionClose)
		m := websocket.FormatCloseMessage(statusCode, "")
		wsc.conn.WriteControl(mt, m, time.Now().Add(time.Second))

		return nil
	})

	conn.SetPingHandler(func(appData string) error {
		if wsc.PingHandler != nil {
			return wsc.PingHandler(appData)
		}

		mt := int(WebSocketMessageTypePong)
		m := []byte(appData)
		err := wsc.conn.WriteControl(mt, m, time.Now().Add(time.Second))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}

		return err
	})

	conn.SetPongHandler(func(appData string) error {
		if wsc.PongHandler != nil {
			return wsc.PongHandler(appData)
		}

		return nil
	})

	return wsc, nil
}

// Flush flushes buffered data to the client.
func (r *Response) Flush() {
	r.writer.(http.Flusher).Flush()
}

// Hijack took over the connection from the server.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.writer.(http.Hijacker).Hijack()
}

// CloseNotify returns a channel that receives at most a single value when the
// connection has gone away.
func (r *Response) CloseNotify() <-chan bool {
	return r.writer.(http.CloseNotifier).CloseNotify()
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

	return r.writer.(http.Pusher).Push(target, pos)
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
func (w *countingWriter) Write(p []byte) (int, error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

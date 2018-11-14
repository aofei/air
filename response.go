package air

import (
	"bytes"
	"crypto/sha256"
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
	netURL "net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/net/html"
)

// Response is an HTTP response.
type Response struct {
	Status        int
	Body          io.Writer
	ContentLength int64
	Written       bool

	request *Request
	writer  http.ResponseWriter
	headers []*Header
	cookies []*Cookie
}

// Header returns the matched `Header` for the name case-insensitively. It
// returns nil if not found.
func (r *Response) Header(name string) *Header {
	name = strings.ToLower(name)
	for _, h := range r.headers {
		if strings.ToLower(h.Name) == name {
			return h
		}
	}

	return nil
}

// Headers returns all the `Header` in the r.
func (r *Response) Headers() []*Header {
	return r.headers
}

// SetHeader sets the `Header` entries associated with the name to the values.
func (r *Response) SetHeader(name string, values ...string) {
	name = strings.ToLower(name)
	if h := r.Header(name); h != nil {
		h.Values = values
	} else if len(values) > 0 {
		r.headers = append(r.headers, &Header{
			Name:   name,
			Values: values,
		})
	}

	if r.Written {
		r.writer.Header()[name] = values
	}
}

// Cookie returns the matched `Cookie` for the name. It returns nil if not
// found.
func (r *Response) Cookie(name string) *Cookie {
	for _, c := range r.cookies {
		if c.Name == name {
			return c
		}
	}

	return nil
}

// Cookies returns all the `Cookie` in the r.
func (r *Response) Cookies() []*Cookie {
	return r.cookies
}

// SetCookie sets the `Cookie` entries associated with the name to the c.
func (r *Response) SetCookie(name string, c *Cookie) {
	if c != nil && c.Name != name {
		return
	}

	for i := range r.cookies {
		if r.cookies[i].Name == name {
			if c != nil {
				r.cookies[i] = c
			} else if i == 0 {
				r.cookies = r.cookies[1:]
			} else if l := len(r.cookies); i == l-1 {
				r.cookies = r.cookies[:l-2]
			} else {
				r.cookies = append(
					r.cookies[:i],
					r.cookies[i+1:]...,
				)
			}

			return
		}
	}

	if c != nil {
		r.cookies = append(r.cookies, c)
	}
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

		if !DebugMode &&
			HTTPSEnforced &&
			theServer.server.TLSConfig != nil &&
			r.Header("strict-transport-security").Value() == "" {
			r.SetHeader(
				"strict-transport-security",
				"max-age=31536000",
			)
		}

		if reader != nil {
			if r.Status >= 200 &&
				r.Status < 300 &&
				r.Header("accept-ranges").Value() == "" {
				r.SetHeader("accept-ranges", "bytes")
			}

			if reader == content && r.ContentLength == 0 {
				r.ContentLength, _ = content.Seek(
					0,
					io.SeekEnd,
				)
				content.Seek(0, io.SeekStart)
			}
		}

		if r.ContentLength >= 0 &&
			r.Header("content-length").Value() == "" &&
			r.Header("transfer-encoding").Value() == "" &&
			r.Status >= 200 &&
			r.Status != 204 &&
			(r.Status >= 300 || r.request.Method != "CONNECT") {
			r.SetHeader(
				"content-length",
				strconv.FormatInt(r.ContentLength, 10),
			)
		} else {
			r.ContentLength = 0
		}

		for _, h := range r.Headers() {
			r.writer.Header()[h.Name] = h.Values
		}

		for _, c := range r.Cookies() {
			if vs := r.writer.Header()["set-cookie"]; len(vs) > 0 {
				vs = append(vs, c.String())
			} else {
				r.writer.Header()["set-cookie"] = []string{
					c.String(),
				}
			}
		}

		r.writer.Header()["server"] = []string{"Air"}

		r.writer.WriteHeader(r.Status)
		if r.request.Method != "HEAD" && reader != nil {
			io.CopyN(r.writer, reader, r.ContentLength)
		}

		r.Written = true
	}()

	if r.Status >= 400 { // Something has gone wrong
		canWrite = true
		return nil
	}

	im := r.request.Header("if-match").Value()
	et := r.Header("etag").Value()
	ius, _ := http.ParseTime(
		r.request.Header("if-unmodified-since").Value(),
	)
	lm, _ := http.ParseTime(r.Header("last-modified").Value())
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

	inm := r.request.Header("if-none-match").Value()
	ims, _ := http.ParseTime(r.request.Header("if-modified-since").Value())
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

			if eTagWeakMatch(eTag, r.Header("etag").Value()) {
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

	if r.Status >= 300 && r.Status < 400 {
		if r.Status == 304 {
			r.SetHeader("content-type")
			r.SetHeader("content-length")
		}

		canWrite = true

		return nil
	} else if r.Status == 412 {
		return errors.New("precondition failed")
	} else if content == nil { // Nothing needs to be written
		canWrite = true
		return nil
	}

	ct := r.Header("content-type").Value()
	if ct == "" {
		// Read a chunk to decide between UTF-8 text and binary.
		b := [1 << 9]byte{}
		n, _ := io.ReadFull(content, b[:])
		ct = http.DetectContentType(b[:n])
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		r.SetHeader("content-type", ct)
	}

	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	} else if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	r.ContentLength = size

	rh := r.request.Header("range").Value()
	if rh == "" {
		canWrite = true
		return nil
	} else if r.request.Method == "GET" || r.request.Method == "HEAD" {
		if ir := r.request.Header("if-range").Value(); ir != "" {
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
			// If no start is specified, end specifies the range
			// start relative to the end of the file.
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
		r.SetHeader("content-range", fmt.Sprintf("bytes */%d", size))
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
		// RFC 2616, section 14.16:
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
		r.SetHeader("content-range", ra.contentRange(size))
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
		r.SetHeader(
			"content-type",
			"multipart/byteranges; boundary="+mw.Boundary(),
		)

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

// WriteBlob responds to the client with the content b.
func (r *Response) WriteBlob(b []byte) error {
	if ct := r.Header("content-type").Value(); ct != "" {
		var err error
		if b, err = theMinifier.minify(ct, b); err != nil {
			return err
		}
	}

	return r.Write(bytes.NewReader(b))
}

// WriteString responds to the client with the "text/plain" content s.
func (r *Response) WriteString(s string) error {
	r.SetHeader("content-type", "text/plain; charset=utf-8")
	return r.WriteBlob([]byte(s))
}

// WriteJSON responds to the client with the "application/json" content v.
func (r *Response) WriteJSON(v interface{}) error {
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

	r.SetHeader("content-type", "application/json; charset=utf-8")

	return r.WriteBlob(b)
}

// WriteXML responds to the client with the "application/xml" content v.
func (r *Response) WriteXML(v interface{}) error {
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

	r.SetHeader("content-type", "application/xml; charset=utf-8")

	return r.WriteBlob(append([]byte(xml.Header), b...))
}

// WriteMsgpack responds to the client with the "application/msgpack" content v.
func (r *Response) WriteMsgpack(v interface{}) error {
	b, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}

	r.SetHeader("content-type", "application/msgpack")

	return r.WriteBlob(b)
}

// WriteProtobuf responds to the client with the "application/protobuf" content
// v.
func (r *Response) WriteProtobuf(v interface{}) error {
	b, err := proto.Marshal(v.(proto.Message))
	if err != nil {
		return err
	}

	r.SetHeader("content-type", "application/protobuf")

	return r.WriteBlob(b)
}

// WriteTOML responds to the client with the "application/toml" content v.
func (r *Response) WriteTOML(v interface{}) error {
	buf := &bytes.Buffer{}
	if err := toml.NewEncoder(buf).Encode(v); err != nil {
		return err
	}

	r.SetHeader("content-type", "application/toml; charset=utf-8")

	return r.WriteBlob(buf.Bytes())
}

// WriteHTML responds to the client with the "text/html" content h.
func (r *Response) WriteHTML(h string) error {
	if AutoPushEnabled && r.request.request.ProtoMajor == 2 {
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

	r.SetHeader("content-type", "text/html; charset=utf-8")

	return r.WriteBlob([]byte(h))
}

// Render renders one or more HTML templates with the m and responds to the
// client with the "text/html" content. The results rendered by the former can
// be inherited by accessing the `m["InheritedHTML"]`.
func (r *Response) Render(m map[string]interface{}, templates ...string) error {
	buf := bytes.Buffer{}
	for _, t := range templates {
		m["InheritedHTML"] = template.HTML(buf.String())
		buf.Reset()
		err := theRenderer.render(&buf, t, m, r.request.LocalizedString)
		if err != nil {
			return err
		}
	}

	return r.WriteHTML(buf.String())
}

// WriteFile responds to the client with a file content with the filename.
func (r *Response) WriteFile(filename string) error {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return err
	} else if fi, err := os.Stat(filename); err != nil {
		return err
	} else if fi.IsDir() {
		if p := r.request.request.URL.EscapedPath(); !hasLastSlash(p) {
			p = path.Base(p) + "/"
			if q := r.request.request.URL.RawQuery; q != "" {
				p += "?" + q
			}

			r.Status = 301

			return r.Redirect(p)
		}

		filename += "index.html"
	}

	var (
		c  io.ReadSeeker
		ct string
		et []byte
		mt time.Time
	)
	if a, err := theCoffer.asset(filename); err != nil {
		return err
	} else if a != nil {
		c = bytes.NewReader(a.content)
		ct = a.mimeType
		et = a.checksum[:]
		mt = a.modTime
	} else {
		f, err := os.Open(filename)
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

	if r.Header("content-type").Value() == "" {
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(filename))
		}

		if ct != "" { // Don't worry, someone will check it later
			r.SetHeader("content-type", ct)
		}
	}

	if r.Header("etag").Value() == "" {
		if et == nil {
			h := sha256.New()
			if _, err := io.Copy(h, c); err != nil {
				return err
			}

			et = h.Sum(nil)
		}

		r.SetHeader("etag", fmt.Sprintf(`"%x"`, et))
	}

	if r.Header("last-modified").Value() == "" {
		r.SetHeader("last-modified", mt.UTC().Format(http.TimeFormat))
	}

	return r.Write(c)
}

// Redirect responds to the client with a redirection to the url.
func (r *Response) Redirect(url string) error {
	if r.Status < 300 || r.Status >= 400 {
		r.Status = 302
	}

	// If the url was relative, make its path absolute by combining with the
	// request path. The client would probably do this for us, but doing it
	// ourselves is more reliable.
	// See RFC 7231, section 7.1.2.
	if u, err := netURL.Parse(url); err != nil {
		return err
	} else if u.Scheme == "" && u.Host == "" {
		if url == "" || url[0] != '/' {
			// Make relative path absolute.
			od, _ := path.Split(r.request.request.URL.Path)
			url = od + url
		}

		query := ""
		if i := strings.Index(url, "?"); i != -1 {
			url, query = url[:i], url[i:]
		}

		// Clean up but preserve trailing slash.
		trailing := strings.HasSuffix(url, "/")
		url = path.Clean(url)
		if trailing && !strings.HasSuffix(url, "/") {
			url += "/"
		}

		url += query
	}

	r.SetHeader("location", url)
	if r.Header("content-type").Value() != "" {
		return r.Write(nil)
	}

	// RFC 7231 notes that a short HTML body is usually included in the
	// response because older user agents may not understand status 301 and
	// 307.
	if r.request.Method == "GET" || r.request.Method == "HEAD" {
		r.SetHeader("content-type", "text/html; charset=utf-8")
	}

	// Shouldn't send the body for POST or HEAD; that leaves GET.
	var body io.ReadSeeker
	if r.request.Method == "GET" {
		body = strings.NewReader(fmt.Sprintf(
			"<a href=\"%s\">%s</a>.\n",
			template.HTMLEscapeString(url),
			strings.ToLower(http.StatusText(r.Status)),
		))
	}

	return r.Write(body)
}

// WebSocket switches the connection to the WebSocket protocol.
func (r *Response) WebSocket() (*WebSocket, error) {
	r.Status = 101

	for _, h := range r.Headers() {
		r.writer.Header()[h.Name] = h.Values
	}

	for _, c := range r.Cookies() {
		if vs := r.writer.Header()["set-cookie"]; len(vs) > 0 {
			vs = append(vs, c.String())
		} else {
			r.writer.Header()["set-cookie"] = []string{
				c.String(),
			}
		}
	}

	r.Written = true

	wsu := &websocket.Upgrader{
		HandshakeTimeout: WebSocketHandshakeTimeout,
		Error: func(
			_ http.ResponseWriter,
			_ *http.Request,
			status int,
			_ error,
		) {
			r.Status = status

			for k := range r.writer.Header() {
				delete(r.writer.Header(), k)
			}

			r.Written = false
		},
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}
	if len(WebSocketSubprotocols) > 0 {
		wsu.Subprotocols = WebSocketSubprotocols
	}

	conn, err := wsu.Upgrade(r.writer, r.request.request, r.writer.Header())
	if err != nil {
		return nil, err
	}

	ws := &WebSocket{
		conn: conn,
	}

	conn.SetCloseHandler(func(statusCode int, reason string) error {
		ws.closed = true

		if ws.ConnectionCloseHandler != nil {
			return ws.ConnectionCloseHandler(statusCode, reason)
		}

		conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(statusCode, ""),
			time.Now().Add(time.Second),
		)

		return nil
	})

	conn.SetPingHandler(func(appData string) error {
		if ws.PingHandler != nil {
			return ws.PingHandler(appData)
		}

		err := conn.WriteControl(
			websocket.PongMessage,
			[]byte(appData),
			time.Now().Add(time.Second),
		)
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}

		return err
	})

	conn.SetPongHandler(func(appData string) error {
		if ws.PongHandler != nil {
			return ws.PongHandler(appData)
		}

		return nil
	})

	return ws, nil
}

// Push initiates an HTTP/2 server push. This constructs a synthetic request
// using the target and headers, serializes that request into a PUSH_PROMISE
// frame, then dispatches that request using the server's request handler. The
// target must either be an absolute path (like "/path") or an absolute URL
// that contains a valid host and the same scheme as the parent request. If the
// target is a path, it will inherit the scheme and host of the parent request.
// The headers specifies additional promised request headers. The headers
// cannot include HTTP/2 pseudo headers like ":path" and ":scheme", which
// will be added automatically.
func (r *Response) Push(target string, headers []*Header) error {
	p, ok := r.writer.(http.Pusher)
	if !ok {
		return nil
	}

	var pos *http.PushOptions
	if l := len(headers); l > 0 {
		pos = &http.PushOptions{
			Method: "GET",
			Header: make(http.Header, l),
		}
		for _, h := range headers {
			pos.Header[h.Name] = h.Values
		}
	}

	return p.Push(target, pos)
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
	// See RFC 7232, section 2.3.
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

// eTagStrongMatch reports whether the a and the b match using strong ETag
// comparison.
func eTagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

// eTagWeakMatch reports whether the a and the b match using weak ETag
// comparison.
func eTagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

// contentRange return a Content-Range header of the r.
func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

// header return  the MIME header of the r.
func (r httpRange) header(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"content-range": {r.contentRange(size)},
		"content-type":  {contentType},
	}
}

// countingWriter counts how many bytes have been written to it.
type countingWriter int64

// Write implements the `io.Writer`.
func (w *countingWriter) Write(p []byte) (int, error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

// responseBody provides a convenient way to continuously write content to the
// client.
type responseBody struct {
	response *Response
}

// Write implements the `io.Writer`.
func (rb *responseBody) Write(b []byte) (int, error) {
	if !rb.response.Written {
		if err := rb.response.Write(nil); err != nil {
			return 0, err
		}
	}

	n, err := rb.response.writer.Write(b)
	rb.response.ContentLength += int64(n)

	return n, err
}

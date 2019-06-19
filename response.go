package air

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aofei/mimesniffer"
	"github.com/cespare/xxhash/v2"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/net/html"
	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	yaml "gopkg.in/yaml.v2"
)

// Response is an HTTP response.
//
// The `Response` not only represents HTTP/1.x responses, but also represents
// HTTP/2 responses, and always acts as HTTP/2 responses.
type Response struct {
	// Air is where the current response belong.
	Air *Air

	// Status is the status code giving the result of the attempt to
	// understand and satisfy the request.
	//
	// See RFC 7231, section 6.
	//
	// For HTTP/1.x, it will be put in the response-line.
	//
	// For HTTP/2, it will be the ":status" pseudo-header.
	//
	// E.g.: 200
	Status int

	// Header is the header map of the current response.
	//
	// By setting the Trailer header to the names of the trailers which will
	// come later. In this case, those names of the header map are treated
	// as if they were trailers.
	//
	// See RFC 7231, section 7.
	//
	// The `Header` is basically the same for both HTTP/1.x and HTTP/2. The
	// only difference is that HTTP/2 requires header names to be lowercase
	// (for aesthetic reasons, this framework decided to follow this rule
	// implicitly, so please use the header name in the HTTP/1.x way). See
	// RFC 7540, section 8.1.2.
	//
	// E.g.: {"Foo": ["bar"]}
	Header http.Header

	// Body is the message body of the current response. It can be used to
	// write a streaming response.
	Body io.Writer

	// ContentLength records the length of the associated content. The value
	// -1 indicates that the length is unknown (it will continue to increase
	// as the data written to the `Body` increases). Values >= 0 indicate
	// that the given number of bytes has been written to the `Body`.
	ContentLength int64

	// Written indicates whether the current response has been written.
	Written bool

	// Minified indicates whether the message body of the current response
	// has been minifed.
	Minified bool

	// Gzipped indicates whether the message body of the current response
	// has been gzipped.
	Gzipped bool

	req               *Request
	hrw               http.ResponseWriter
	ohrw              http.ResponseWriter
	servingContent    bool
	serveContentError error
	reverseProxying   bool
	deferredFuncs     []func()
}

// HTTPResponseWriter returns the underlying `http.ResponseWriter` of the r.
//
// ATTENTION: You should never call this method unless you know what you are
// doing. And, be sure to call the `SetHTTPResponseWriter` of the r when you
// have modified it.
func (r *Response) HTTPResponseWriter() http.ResponseWriter {
	return r.hrw
}

// SetHTTPResponseWriter sets the hrw to the underlying `http.ResponseWriter` of
// the r.
//
// ATTENTION: You should never call this method unless you know what you are
// doing.
func (r *Response) SetHTTPResponseWriter(hrw http.ResponseWriter) {
	r.Header = hrw.Header()
	r.Body = hrw
	r.hrw = hrw
}

// SetCookie sets the c to the `Header` of the r. Invalid cookies will be
// silently dropped.
func (r *Response) SetCookie(c *http.Cookie) {
	if v := c.String(); v != "" {
		r.Header.Add("Set-Cookie", v)
	}
}

// Write writes the content to the client.
//
// The main benefit of the `Write` over the `io.Copy` with the `Body` of the r
// is that it handles range requests properly, sets the Content-Type header, and
// handles the If-Match header, the If-Unmodified-Since header, the
// If-None-Match header, the If-Modified-Since header and the If-Range header of
// the requests.
func (r *Response) Write(content io.ReadSeeker) error {
	if content == nil { // No content, no benefit
		if !r.Written {
			r.hrw.WriteHeader(r.Status)
		}

		return nil
	}

	if r.Written {
		if r.req.Method != http.MethodHead {
			io.Copy(r.hrw, content)
		}

		return nil
	}

	if r.Header.Get("Content-Type") == "" {
		b := r.Air.contentTypeSnifferBufferPool.Get().([]byte)
		defer r.Air.contentTypeSnifferBufferPool.Put(b)

		n, err := io.ReadFull(content, b)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}

		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		r.Header.Set("Content-Type", mimesniffer.Sniff(b[:n]))
	}

	if !r.Minified && r.Air.MinifierEnabled {
		mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if stringSliceContainsCIly(r.Air.MinifierMIMETypes, mt) {
			b, err := ioutil.ReadAll(content)
			if err != nil {
				return err
			}

			if b, err = r.Air.minifier.minify(mt, b); err != nil {
				return err
			}

			content = bytes.NewReader(b)
			r.Minified = true
			defer func() {
				if !r.Written {
					r.Minified = false
				}
			}()
		}
	}

	if r.Status < http.StatusBadRequest {
		lm := time.Time{}
		if lmh := r.Header.Get("Last-Modified"); lmh != "" {
			lm, _ = http.ParseTime(lmh)
		}

		r.servingContent = true
		r.serveContentError = nil
		http.ServeContent(r.hrw, r.req.HTTPRequest(), "", lm, content)
		r.servingContent = false

		return r.serveContentError
	}

	if r.Header.Get("Content-Encoding") == "" {
		cl, err := content.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}

		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		r.Header.Set("Content-Length", strconv.FormatInt(cl, 10))
	}

	r.Header.Del("ETag")
	r.Header.Del("Last-Modified")

	if r.req.Method == http.MethodHead {
		r.hrw.WriteHeader(r.Status)
	} else {
		io.Copy(r.hrw, content)
	}

	return nil
}

// WriteString writes the s as a "text/plain" content to the client.
func (r *Response) WriteString(s string) error {
	r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return r.Write(strings.NewReader(s))
}

// WriteJSON writes an "application/json" content encoded from the v to the
// client.
func (r *Response) WriteJSON(v interface{}) error {
	var (
		b   []byte
		err error
	)

	if r.Air.DebugMode {
		b, err = json.MarshalIndent(v, "", "\t")
	} else {
		b, err = json.Marshal(v)
	}

	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	return r.Write(bytes.NewReader(b))
}

// WriteXML writes an "application/xml" content encoded from the v to the
// client.
func (r *Response) WriteXML(v interface{}) error {
	var (
		b   []byte
		err error
	)

	if r.Air.DebugMode {
		b, err = xml.MarshalIndent(v, "", "\t")
	} else {
		b, err = xml.Marshal(v)
	}

	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/xml; charset=utf-8")

	return r.Write(strings.NewReader(xml.Header + string(b)))
}

// WriteProtobuf writes an "application/protobuf" content encoded from the v to
// the client.
func (r *Response) WriteProtobuf(v interface{}) error {
	b, err := proto.Marshal(v.(proto.Message))
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/protobuf")

	return r.Write(bytes.NewReader(b))
}

// WriteMsgpack writes an "application/msgpack" content encoded from the v to
// the client.
func (r *Response) WriteMsgpack(v interface{}) error {
	b, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/msgpack")

	return r.Write(bytes.NewReader(b))
}

// WriteTOML writes an "application/toml" content encoded from the v to the
// client.
func (r *Response) WriteTOML(v interface{}) error {
	buf := bytes.Buffer{}
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/toml; charset=utf-8")

	return r.Write(bytes.NewReader(buf.Bytes()))
}

// WriteYAML writes an "application/yaml" content encoded from the v to the
// client.
func (r *Response) WriteYAML(v interface{}) error {
	buf := bytes.Buffer{}
	if err := yaml.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/yaml; charset=utf-8")

	return r.Write(bytes.NewReader(buf.Bytes()))
}

// WriteHTML writes the h as a "text/html" content to the client.
func (r *Response) WriteHTML(h string) error {
	if r.Air.AutoPushEnabled && r.req.HTTPRequest().ProtoMajor == 2 {
		tree, err := html.Parse(strings.NewReader(h))
		if err != nil {
			return err
		}

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				avoid, target := false, ""
				switch strings.ToLower(n.Data) {
				case "link":
					relChecked := false
				LinkLoop:
					for _, a := range n.Attr {
						switch strings.ToLower(a.Key) {
						case "rel":
							switch strings.ToLower(
								a.Val,
							) {
							case "preload", "icon":
								avoid = true
								break LinkLoop
							}

							relChecked = true
						case "href":
							target = a.Val
							if relChecked {
								break LinkLoop
							}
						}
					}
				case "img", "script":
				ImgScriptLoop:
					for _, a := range n.Attr {
						switch strings.ToLower(a.Key) {
						case "src":
							target = a.Val
							break ImgScriptLoop
						}
					}
				}

				if !avoid && path.IsAbs(target) {
					r.Push(target, nil)
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(tree)
	}

	r.Header.Set("Content-Type", "text/html; charset=utf-8")

	return r.Write(strings.NewReader(h))
}

// Render renders one or more HTML templates with the m and writes the results
// as a "text/html" content to the client. The results rendered by the former
// can be inherited by accessing the `m["InheritedHTML"]`.
func (r *Response) Render(m map[string]interface{}, templates ...string) error {
	buf := bytes.Buffer{}
	for _, t := range templates {
		if buf.Len() > 0 {
			if m == nil {
				m = make(map[string]interface{}, 1)
			}

			m["InheritedHTML"] = template.HTML(buf.String())
		}

		buf.Reset()

		err := r.Air.renderer.render(&buf, t, m, r.req.LocalizedString)
		if err != nil {
			return err
		}
	}

	return r.WriteHTML(buf.String())
}

// WriteFile writes a file content targeted by the filename to the client.
func (r *Response) WriteFile(filename string) error {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return err
	} else if fi, err := os.Stat(filename); err != nil {
		return err
	} else if fi.IsDir() {
		p, q := splitPathQuery(r.req.Path)
		if !strings.HasSuffix(p, "/") {
			p = fmt.Sprint(path.Base(p), "/")
			if q != "" {
				p = fmt.Sprint(p, "?", q)
			}

			r.Status = http.StatusMovedPermanently

			return r.Redirect(p)
		}

		filename = fmt.Sprint(filename, "index.html")
	}

	var (
		c  io.ReadSeeker
		ct string
		et []byte
		mt time.Time
	)

	if r.Air.CofferEnabled {
		if a, err := r.Air.coffer.asset(filename); err != nil {
			return err
		} else if a != nil {
			r.Minified = a.minified

			var ac []byte
			if r.Air.GzipEnabled && a.gzippedDigest != nil &&
				httpguts.HeaderValuesContainsToken(
					r.req.Header["Accept-Encoding"],
					"gzip",
				) {
				if ac = a.content(true); ac != nil {
					r.Gzipped = true
				}
			} else {
				ac = a.content(false)
			}

			if ac != nil {
				c = bytes.NewReader(ac)
				ct = a.mimeType
				et = a.digest
				mt = a.modTime
			}
		}
	}

	if c == nil {
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

	if r.Header.Get("Content-Type") == "" {
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(filename))
		}

		r.Header.Set("Content-Type", ct)
	}

	if r.Header.Get("ETag") == "" {
		if et == nil {
			h := xxhash.New()
			if _, err := io.Copy(h, c); err != nil {
				return err
			}

			if _, err := c.Seek(0, io.SeekStart); err != nil {
				return err
			}

			et = h.Sum(nil)
		}

		r.Header.Set("ETag", fmt.Sprintf(
			"%q",
			base64.StdEncoding.EncodeToString(et),
		))
	}

	if r.Header.Get("Last-Modified") == "" {
		r.Header.Set("Last-Modified", mt.UTC().Format(http.TimeFormat))
	}

	return r.Write(c)
}

// Redirect writes the url as a redirection to the client. Note that the
// `Status` of the r will be the `http.StatusFound` if it is not a redirection
// status.
func (r *Response) Redirect(url string) error {
	if r.Status < http.StatusMultipleChoices ||
		r.Status >= http.StatusBadRequest {
		r.Status = http.StatusFound
	}

	http.Redirect(r.hrw, r.req.HTTPRequest(), url, r.Status)

	return nil
}

// WebSocket switches the connection of the r to the WebSocket protocol. See RFC
// 6455.
func (r *Response) WebSocket() (*WebSocket, error) {
	r.Status = http.StatusSwitchingProtocols
	r.Written = true

	wsu := &websocket.Upgrader{
		HandshakeTimeout: r.Air.WebSocketHandshakeTimeout,
		Error: func(
			_ http.ResponseWriter,
			_ *http.Request,
			status int,
			_ error,
		) {
			r.Status = status
			r.Written = false
		},
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}
	if len(r.Air.WebSocketSubprotocols) > 0 {
		wsu.Subprotocols = r.Air.WebSocketSubprotocols
	}

	conn, err := wsu.Upgrade(r.ohrw, r.req.HTTPRequest(), r.Header)
	if err != nil {
		return nil, err
	}

	ws := &WebSocket{
		conn: conn,
	}

	conn.SetCloseHandler(func(status int, reason string) error {
		ws.Closed = true

		if ws.ConnectionCloseHandler != nil {
			return ws.ConnectionCloseHandler(status, reason)
		}

		conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(status, ""),
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
// using the target and the pos, serializes that request into a "PUSH_PROMISE"
// frame, then dispatches that request using the server's request handler. If
// pos is nil, default options are used.
//
// The target must either be an absolute path (like "/path") or an absolute URL
// that contains a valid authority and the same scheme as the parent request. If
// the target is a path, it will inherit the scheme and authority of the parent
// request.
//
// It returns `http.ErrNotSupported` if the client has disabled push or if push
// is not supported on the underlying connection.
func (r *Response) Push(target string, pos *http.PushOptions) error {
	p, ok := r.ohrw.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}

	return p.Push(target, pos)
}

// ProxyPass passes the request to the target and writes the response from the
// target to the client by using the reverse proxy technique. If the rpm is
// non-nil, then it will be used to modify the request to the target and the
// response from the target.
//
// The target must be based on the HTTP protocol (such as HTTP(S), WebSocket and
// gRPC). So, the scheme of the target must be "http", "https", "ws", "wss",
// "grpc" or "grpcs".
func (r *Response) ProxyPass(target string, rpm *ReverseProxyModifier) error {
	if r.Written {
		return errors.New("air: request has been written")
	}

	if rpm == nil {
		rpm = &ReverseProxyModifier{}
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return err
	}

	targetURL.Scheme = strings.ToLower(targetURL.Scheme)
	switch targetURL.Scheme {
	case "http", "https", "ws", "wss", "grpc", "grpcs":
	default:
		return fmt.Errorf(
			"air: unsupported reverse proxy scheme: %s",
			targetURL.Scheme,
		)
	}

	targetURL.Host = strings.ToLower(targetURL.Host)

	reqPath, reqQuery := splitPathQuery(r.req.Path)
	targetURL.Path = path.Join(targetURL.Path, reqPath)
	if targetURL.RawQuery == "" || reqQuery == "" {
		targetURL.RawQuery = fmt.Sprint(targetURL.RawQuery, reqQuery)
	} else {
		targetURL.RawQuery = fmt.Sprint(
			targetURL.RawQuery,
			"&",
			reqQuery,
		)
	}

	targetPathQuery := targetURL.Path
	if targetURL.RawQuery != "" {
		targetPathQuery = fmt.Sprint(
			targetPathQuery,
			"?",
			targetURL.RawQuery,
		)
	}

	if mrp := rpm.ModifyRequestPath; mrp != nil {
		mp := mrp(targetPathQuery)
		if mp != targetPathQuery {
			targetURL.Path, targetURL.RawQuery = splitPathQuery(mp)
		}
	}

	targetHeader := make(http.Header, len(r.req.Header))
	for n, vs := range r.req.Header {
		targetHeader[n] = make([]string, len(vs))
		copy(targetHeader[n], vs)
	}

	if mrh := rpm.ModifyRequestHeader; mrh != nil {
		mrh(targetHeader)
	}

	if _, ok := targetHeader["User-Agent"]; !ok {
		// Explicitly disable the User-Agent header so it's not set to
		// default value.
		targetHeader.Set("User-Agent", "")
	}

	targetBody := r.req.Body
	if mrb := rpm.ModifyRequestBody; mrb != nil {
		targetBody, err = mrb(r.req.Body)
		if err != nil {
			return err
		}
	}

	switch targetURL.Scheme {
	case "ws", "wss":
	default:
		var reverseProxyError error
		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				if mrm := rpm.ModifyRequestMethod; mrm != nil {
					req.Method = mrm(req.Method)
				}

				req.URL = targetURL
				req.Header = targetHeader
				req.Body = ioutil.NopCloser(targetBody)
			},
			FlushInterval: 100 * time.Millisecond,
			Transport:     r.Air.reverseProxyTransport,
			ErrorLog:      r.Air.errorLogger,
			BufferPool:    r.Air.reverseProxyBufferPool,
			ModifyResponse: func(res *http.Response) error {
				if mrs := rpm.ModifyResponseStatus; mrs != nil {
					res.StatusCode = mrs(res.StatusCode)
				}

				if mrh := rpm.ModifyResponseHeader; mrh != nil {
					mrh(res.Header)
				}

				if httpguts.HeaderValuesContainsToken(
					res.Header["Content-Encoding"],
					"gzip",
				) {
					r.Gzipped = true
				}

				if mrb := rpm.ModifyResponseBody; mrb != nil {
					b, err := mrb(res.Body)
					if err != nil {
						return err
					}

					res.Body = b
				}

				return nil
			},
			ErrorHandler: func(
				_ http.ResponseWriter,
				_ *http.Request,
				err error,
			) {
				r.Status = http.StatusBadGateway
				reverseProxyError = err
			},
		}
		switch targetURL.Scheme {
		case "grpc", "grpcs":
			rp.FlushInterval /= 100 // For gRPC streaming
		}

		r.reverseProxying = true
		rp.ServeHTTP(r.hrw, r.req.HTTPRequest())
		r.reverseProxying = false

		return reverseProxyError
	}

	targetHeader.Del("Upgrade")
	targetHeader.Del("Connection")
	targetHeader.Del("Keep-Alive")
	targetHeader.Del("TE")
	targetHeader.Del("Trailer")
	targetHeader.Del("Sec-WebSocket-Key")
	targetHeader.Del("Sec-WebSocket-Extensions")
	targetHeader.Del("Sec-WebSocket-Accept")
	targetHeader.Del("Sec-WebSocket-Version")

	dc, res, err := websocket.DefaultDialer.Dial(
		targetURL.String(),
		targetHeader,
	)
	if err != nil {
		r.Status = http.StatusBadGateway
		return err
	}
	defer dc.Close()

	res.Header.Del("Upgrade")
	res.Header.Del("Connection")
	res.Header.Del("Keep-Alive")
	res.Header.Del("Transfer-Encoding")
	res.Header.Del("Trailer")
	res.Header.Del("Sec-WebSocket-Extensions")
	res.Header.Del("Sec-WebSocket-Accept")

	for n, vs := range res.Header {
		for _, v := range vs {
			targetHeader.Add(n, v)
		}
	}

	wsu := &websocket.Upgrader{
		HandshakeTimeout: r.Air.WebSocketHandshakeTimeout,
		Error: func(
			_ http.ResponseWriter,
			_ *http.Request,
			status int,
			_ error,
		) {
			r.Status = status
		},
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}
	if len(r.Air.WebSocketSubprotocols) > 0 {
		wsu.Subprotocols = r.Air.WebSocketSubprotocols
	}

	uc, err := wsu.Upgrade(r.ohrw, r.req.HTTPRequest(), r.Header)
	if err != nil {
		return err
	}
	defer uc.Close()

	r.Status = http.StatusSwitchingProtocols
	r.Written = true

	errChan := make(chan error, 1)
	go replicateWebSocketConn(uc, dc, errChan)
	go replicateWebSocketConn(dc, uc, errChan)

	err = <-errChan
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		err = nil
	}

	return err
}

// Defer pushes the f onto the stack of functions that will be called after
// responding. Nil functions will be silently dropped.
func (r *Response) Defer(f func()) {
	if f != nil {
		r.deferredFuncs = append(r.deferredFuncs, f)
	}
}

// ReverseProxyModifier is used by the `Response.ProxyPass` to modify the
// request to the target and the response from the traget.
//
// Note that any field in the `ReverseProxyModifier` can be nil, which means
// there is no need to modify that value.
type ReverseProxyModifier struct {
	// ModifyRequestMethod modifies the method of the request to the target.
	ModifyRequestMethod func(method string) string

	// ModifyRequestPath modifies the path of the request to the target.
	//
	// Note that the path contains the query part (anyway, the HTTP/2
	// specification says so). Therefore, the returned path must also be in
	// this format.
	ModifyRequestPath func(path string) string

	// ModifyRequestHeader modifies the header of the request to the target.
	ModifyRequestHeader func(header http.Header)

	// ModifyRequestBody modifies the body of the request from the target.
	ModifyRequestBody func(body io.Reader) (io.Reader, error)

	// ModifyResponseStatus modifies the status of the response from the
	// target.
	ModifyResponseStatus func(status int) int

	// ModifyResponseHeader modifies the header of the response from the
	// target.
	ModifyResponseHeader func(header http.Header)

	// ModifyResponseBody modifies the body of the response from the target.
	//
	// It is the caller's responsibility to close the returned
	// `io.ReadCloser`, which means that the `Response.ProxyPass` will be
	/// responsible for closing it.
	ModifyResponseBody func(body io.ReadCloser) (io.ReadCloser, error)
}

// responseWriter used to tie the `Response` and the `http.ResponseWriter`
// together.
type responseWriter struct {
	sync.Mutex

	r     *Response
	w     http.ResponseWriter
	gw    *gzip.Writer
	gwn   int
	b64wc io.WriteCloser
}

// Header implements the `http.ResponseWriter`.
func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

// WriteHeader implements the `http.ResponseWriter`.
func (rw *responseWriter) WriteHeader(status int) {
	rw.Lock()
	defer rw.Unlock()

	if rw.r.Written {
		return
	}

	if rw.r.servingContent {
		if status == http.StatusOK {
			status = rw.r.Status
		} else if status >= http.StatusBadRequest {
			rw.r.Status = status
			rw.r.Header.Del("Content-Type")
			rw.r.Header.Del("X-Content-Type-Options")
			return
		}
	}

	rw.handleGzip()
	rw.handleReverseProxy()
	rw.w.WriteHeader(status)

	rw.r.Status = status
	rw.r.ContentLength = 0
	rw.r.Written = true
}

// Write implements the `http.ResponseWriter`.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.r.Written {
		rw.WriteHeader(rw.r.Status)
	}

	rw.Lock()
	defer rw.Unlock()

	if rw.r.servingContent && rw.r.Status >= http.StatusBadRequest {
		rw.r.serveContentError = errors.New(string(b))
		return 0, nil
	}

	w := io.Writer(rw.w)
	if rw.b64wc != nil {
		w = rw.b64wc
	} else if rw.gw != nil {
		w = rw.gw
	}

	n, err := w.Write(b)
	if n > 0 {
		rw.r.ContentLength += int64(n)
		if w == rw.gw && rw.r.Air.GzipFlushThreshold > 0 {
			rw.gwn += n
			if rw.gwn >= rw.r.Air.GzipFlushThreshold {
				rw.gwn = 0
				rw.gw.Flush()
			}
		}
	}

	return n, err
}

// Flush implements the `http.Flusher`.
func (rw *responseWriter) Flush() {
	if rw.b64wc != nil {
		rw.b64wc.Close()

		w := io.Writer(rw.w)
		if rw.gw != nil {
			w = rw.gw
		}

		rw.b64wc = base64.NewEncoder(base64.StdEncoding, w)
	}

	if rw.gw != nil {
		rw.gw.Flush()
	}

	rw.w.(http.Flusher).Flush()
}

// Push implements the `http.Pusher`.
func (rw *responseWriter) Push(target string, pos *http.PushOptions) error {
	p, ok := rw.w.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}

	return p.Push(target, pos)
}

// handleGzip handles the gzip feature for the rw.
func (rw *responseWriter) handleGzip() {
	if !rw.r.Air.GzipEnabled {
		return
	}

	if !rw.r.Gzipped {
		if cl, _ := strconv.ParseInt(
			rw.r.Header.Get("Content-Length"),
			10,
			64,
		); cl < rw.r.Air.GzipMinContentLength {
			return
		}

		if mt, _, _ := mime.ParseMediaType(
			rw.r.Header.Get("Content-Type"),
		); !stringSliceContainsCIly(rw.r.Air.GzipMIMETypes, mt) {
			return
		}

		if httpguts.HeaderValuesContainsToken(
			rw.r.req.Header["Accept-Encoding"],
			"gzip",
		) {
			rw.gw, _ = rw.r.Air.gzipWriterPool.Get().(*gzip.Writer)
			if rw.gw == nil {
				return
			}

			rw.gw.Reset(rw.w)
			rw.r.Defer(func() {
				if rw.gwn == 0 {
					rw.gw.Reset(ioutil.Discard)
				}

				rw.gw.Close()
				rw.r.Air.gzipWriterPool.Put(rw.gw)
				rw.gw = nil
			})

			rw.r.Gzipped = true
		}
	}

	if rw.r.Gzipped {
		if !httpguts.HeaderValuesContainsToken(
			rw.r.Header["Content-Encoding"],
			"gzip",
		) {
			rw.r.Header.Add("Content-Encoding", "gzip")
		}

		rw.r.Header.Del("Content-Length")

		if et := rw.r.Header.Get("ETag"); et != "" &&
			!strings.HasPrefix(et, "W/") {
			rw.r.Header.Set("ETag", fmt.Sprint("W/", et))
		}
	}

	if !httpguts.HeaderValuesContainsToken(
		rw.r.Header["Vary"],
		"Accept-Encoding",
	) {
		rw.r.Header.Add("Vary", "Accept-Encoding")
	}
}

// handleReverseProxy handles the reverse proxy feature for the rw.
func (rw *responseWriter) handleReverseProxy() {
	if !rw.r.reverseProxying {
		return
	}

	reqct := rw.r.req.Header.Get("Content-Type")
	if !strings.HasPrefix(reqct, "application/grpc-web") {
		return
	}

	reqmt := "application/grpc-web-text"
	if strings.HasSuffix(reqct, reqmt) {
		w := io.Writer(rw.w)
		if rw.gw != nil {
			w = rw.gw
		}

		rw.b64wc = base64.NewEncoder(base64.StdEncoding, w)
	} else {
		reqmt = "application/grpc-web"
	}

	rw.r.Header.Set("Content-Type", strings.Replace(
		rw.r.Header.Get("Content-Type"),
		"application/grpc",
		reqmt,
		1,
	))

	tns := strings.Split(rw.r.Header.Get("Trailer"), ", ")
	rw.r.Header.Del("Trailer")

	hns := make([]string, 0, len(rw.r.Header))
	for n := range rw.r.Header {
		hns = append(hns, n)
	}

	rw.r.Header.Set(
		"Access-Control-Expose-Headers",
		strings.Join(hns, ", "),
	)

	rw.r.Defer(func() {
		ts := make(http.Header, len(tns))
		for _, tn := range tns {
			ltn := strings.ToLower(tn)
			ts[ltn] = append(ts[ltn], rw.r.Header[tn]...)
			rw.r.Header.Del(tn)
		}

		for n, vs := range rw.r.Header {
			if !strings.HasPrefix(n, http.TrailerPrefix) {
				continue
			}

			ltn := strings.ToLower(n[len(http.TrailerPrefix):])
			ts[ltn] = append(ts[ltn], vs...)
			rw.r.Header.Del(n)
		}

		tb := bytes.Buffer{}
		ts.Write(&tb)

		th := []byte{1 << 7, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(th[1:5], uint32(tb.Len()))

		rw.Write(th)
		rw.Write(tb.Bytes())
		rw.Flush()
	})
}

// newReverseProxyTransport returns a new instance of the `http.Transport` with
// reverse proxy support.
func newReverseProxyTransport() *http.Transport {
	rpt := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		DisableCompression:    true,
		MaxIdleConnsPerHost:   200,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	rpt.RegisterProtocol("grpc", newGRPCTransport(false))
	rpt.RegisterProtocol("grpcs", newGRPCTransport(true))

	return rpt
}

// grpcTransport is a transport with the gRPC support.
type grpcTransport struct {
	*http2.Transport

	tlsed bool
}

// newGRPCTransport returns a new instance of the `grpcTransport` with the
// tlsed.
func newGRPCTransport(tlsed bool) *grpcTransport {
	gt := &grpcTransport{
		Transport: &http2.Transport{},

		tlsed: tlsed,
	}
	if !tlsed {
		gt.DialTLS = func(
			network string,
			address string,
			_ *tls.Config,
		) (net.Conn, error) {
			return net.Dial(network, address)
		}

		gt.AllowHTTP = true
	}

	return gt
}

// RoundTrip implements the `http2.Transport`.
func (gt *grpcTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if gt.tlsed {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}

	ct := req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/grpc-web") {
		mt := "application/grpc-web-text"
		if strings.HasSuffix(ct, mt) {
			req.Body = ioutil.NopCloser(base64.NewDecoder(
				base64.StdEncoding,
				req.Body,
			))
		} else {
			mt = "application/grpc-web"
		}

		req.Header.Set(
			"Content-Type",
			strings.Replace(ct, mt, "application/grpc", 1),
		)
	}

	return gt.Transport.RoundTrip(req)
}

// reverseProxyBufferPool is a buffer pool for the reverse proxy.
type reverseProxyBufferPool struct {
	pool *sync.Pool
}

// newReverseProxyBufferPool returns a new instance of the
// `reverseProxyBufferPool`.
func newReverseProxyBufferPool() *reverseProxyBufferPool {
	return &reverseProxyBufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 32<<20)
			},
		},
	}
}

// Get implements the `httputil.BufferPool`.
func (rpbp *reverseProxyBufferPool) Get() []byte {
	return rpbp.pool.Get().([]byte)
}

// Put implements the `httputil.BufferPool`.
func (rpbp *reverseProxyBufferPool) Put(bytes []byte) {
	rpbp.pool.Put(bytes)
}

// replicateWebSocketConn replicates data from the src to the dst with the
// errChan.
func replicateWebSocketConn(dst, src *websocket.Conn, errChan chan error) {
	fwd := func(messageType int, r io.Reader) error {
		w, err := dst.NextWriter(messageType)
		if err != nil {
			return err
		} else if _, err := io.Copy(w, r); err != nil {
			return err
		}

		return w.Close()
	}

	src.SetPingHandler(func(appData string) error {
		return fwd(websocket.PingMessage, strings.NewReader(appData))
	})

	src.SetPongHandler(func(appData string) error {
		return fwd(websocket.PongMessage, strings.NewReader(appData))
	})

	for {
		mt, r, err := src.NextReader()
		if err != nil {
			errChan <- err

			var m []byte
			if e, ok := err.(*websocket.CloseError); !ok ||
				e.Code == websocket.CloseNoStatusReceived {
				m = websocket.FormatCloseMessage(
					websocket.CloseNormalClosure,
					err.Error(),
				)
			} else if ok &&
				e.Code != websocket.CloseAbnormalClosure &&
				e.Code != websocket.CloseTLSHandshake {
				m = websocket.FormatCloseMessage(e.Code, e.Text)
			}

			if m != nil {
				fwd(websocket.CloseMessage, bytes.NewReader(m))
			}

			break
		}

		if err := fwd(mt, r); err != nil {
			errChan <- err
			break
		}
	}
}

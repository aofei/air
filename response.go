package air

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
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
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/cespare/xxhash"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/net/html"
)

// Response is an HTTP response.
type Response struct {
	// Air is where the current response belong.
	Air *Air

	// Status is the status code giving the result of the attempt to
	// understand and satisfy the request.
	//
	// See RFC 7231, section 6.
	//
	// For HTTP/1.x, it will be put in response-line.
	//
	// For HTTP/2, it will be the ":status" pseudo-header.
	Status int

	// Header is the header key-value pair map of the current response.
	//
	// See RFC 7231, section 7.
	//
	// It is basically the same for HTTP/1.x and HTTP/2. The only difference
	// is that HTTP/2 requires header names to be lowercase. See RFC 7540,
	// section 8.1.2.
	Header http.Header

	// Body is the message body of the current response. It can be used to
	// write a streaming response.
	Body io.Writer

	// ContentLength records the length of the bytes that has been written.
	ContentLength int64

	// Written indicates whether the current response has been written.
	Written bool

	// Minified indicates whether the message body of the current response
	// has been minifed.
	Minified bool

	// Gzipped indicates whether the message body of the current response
	// has been gzipped.
	Gzipped bool

	req           *Request
	hrw           http.ResponseWriter
	ohrw          http.ResponseWriter
	deferredFuncs []func()
}

// HTTPResponseWriter returns the underlying `http.ResponseWriter` of the r.
//
// ATTENTION: You should never call this method unless you know what you are
// doing. And, be sure to call the `r#SetHTTPResponseWriter()` when you have
// modified it.
func (r *Response) HTTPResponseWriter() http.ResponseWriter {
	return r.hrw
}

// SetHTTPResponseWriter sets the hrw to the r's underlying
// `http.ResponseWriter`.
//
// ATTENTION: You should never call this method unless you know what you are
// doing.
func (r *Response) SetHTTPResponseWriter(hrw http.ResponseWriter) {
	r.Header = hrw.Header()
	r.Body = hrw
	r.hrw = hrw
}

// SetCookie sets the c to the `r#Header`. Invalid cookies will be silently
// dropped.
func (r *Response) SetCookie(c *http.Cookie) {
	if v := c.String(); v != "" {
		r.Header.Add("Set-Cookie", v)
	}
}

// Write responds to the client with the content.
func (r *Response) Write(content io.ReadSeeker) error {
	if r.Written {
		var err error
		if r.req.Method != http.MethodHead {
			_, err = io.Copy(r.hrw, content)
		}

		return err
	}

	if !r.Minified && r.Air.MinifierEnabled {
		mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if stringSliceContains(r.Air.MinifierMIMETypes, mt) {
			b, err := ioutil.ReadAll(content)
			if err != nil {
				return err
			}

			if b, err = r.Air.minifier.minify(mt, b); err != nil {
				return err
			}

			content = bytes.NewReader(b)
			r.Minified = true
		}
	}

	lm, _ := http.ParseTime(r.Header.Get("Last-Modified"))
	http.ServeContent(r.hrw, r.req.HTTPRequest(), "", lm, content)

	return nil
}

// WriteString responds to the client with the "text/plain" content s.
func (r *Response) WriteString(s string) error {
	r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return r.Write(strings.NewReader(s))
}

// WriteJSON responds to the client with the "application/json" content v.
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

// WriteXML responds to the client with the "application/xml" content v.
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

// WriteMsgpack responds to the client with the "application/msgpack" content v.
func (r *Response) WriteMsgpack(v interface{}) error {
	b, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/msgpack")

	return r.Write(bytes.NewReader(b))
}

// WriteProtobuf responds to the client with the "application/protobuf" content
// v.
func (r *Response) WriteProtobuf(v interface{}) error {
	b, err := proto.Marshal(v.(proto.Message))
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/protobuf")

	return r.Write(bytes.NewReader(b))
}

// WriteTOML responds to the client with the "application/toml" content v.
func (r *Response) WriteTOML(v interface{}) error {
	buf := bytes.Buffer{}
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/toml; charset=utf-8")

	return r.Write(bytes.NewReader(buf.Bytes()))
}

// WriteHTML responds to the client with the "text/html" content h.
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
				switch n.Data {
				case "link":
					relChecked := false
				LinkLoop:
					for _, a := range n.Attr {
						switch strings.ToLower(a.Key) {
						case "rel":
							if v := strings.ToLower(
								a.Val,
							); v == "preload" ||
								v == "icon" {
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

// Render renders one or more HTML templates with the m and responds to the
// client with the "text/html" content. The results rendered by the former can
// be inherited by accessing the `m["InheritedHTML"]`.
func (r *Response) Render(m map[string]interface{}, templates ...string) error {
	buf := bytes.Buffer{}
	for _, t := range templates {
		m["InheritedHTML"] = template.HTML(buf.String())
		buf.Reset()
		err := r.Air.renderer.render(&buf, t, m, r.req.LocalizedString)
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
		hru := r.req.HTTPRequest().URL
		if p := hru.EscapedPath(); !hasLastSlash(p) {
			p = path.Base(p) + "/"
			if q := hru.RawQuery; q != "" {
				p += "?" + q
			}

			r.Status = http.StatusMovedPermanently

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

	if r.Air.CofferEnabled {
		if a, err := r.Air.coffer.asset(filename); err != nil {
			return err
		} else if a != nil {
			r.Minified = a.minified

			var ac []byte
			if r.Air.GzipEnabled &&
				a.gzippedDigest != nil &&
				strings.Contains(
					r.req.Header.Get("Accept-Encoding"),
					"gzip",
				) {
				ac = a.content(true)
				if ac != nil {
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

		if ct != "" {
			r.Header.Set("Content-Type", ct)
		} else {
			r.Header.Del("Content-Type") // Make it sniffable
		}
	}

	if r.Header.Get("ETag") == "" {
		if et == nil {
			h := xxhash.New()
			if _, err := io.Copy(h, c); err != nil {
				return err
			}

			et = h.Sum(nil)
		}

		r.Header.Set(
			"ETag",
			"\""+base64.StdEncoding.EncodeToString(et)+"\"",
		)
	}

	if r.Header.Get("Last-Modified") == "" {
		r.Header.Set("Last-Modified", mt.UTC().Format(http.TimeFormat))
	}

	return r.Write(c)
}

// Redirect responds to the client with a redirection to the url.
func (r *Response) Redirect(url string) error {
	if r.Status < http.StatusMultipleChoices ||
		r.Status >= http.StatusBadRequest {
		r.Status = http.StatusFound
	}

	http.Redirect(r.hrw, r.req.HTTPRequest(), url, r.Status)

	return nil
}

// WebSocket switches the connection to the WebSocket protocol.
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
		ws.closed = true

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
// using the target and the pos, serializes that request into a PUSH_PROMISE
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

// ProxyPass passes the request to the target and responds to the client by
// using the reverse proxy technique.
//
// The target must be based on the HTTP protocol (such as HTTP(S), WebSocket and
// gRPC). So, the scheme of the target must be "http", "https", "ws", "wss",
// "grpc" or "grpcs".
func (r *Response) ProxyPass(target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return err
	}

	rp := httputil.NewSingleHostReverseProxy(u)
	rp.Transport = r.Air.reverseProxyTransport
	rp.ErrorLog = r.Air.errorLogger
	rp.BufferPool = r.Air.reverseProxyBufferPool

	switch u.Scheme {
	case "http", "https":
		rp.FlushInterval = 100 * time.Millisecond
	case "ws", "wss", "grpc", "grpcs":
		rp.FlushInterval = time.Millisecond
	default:
		return errors.New("unsupported reverse proxy scheme")
	}

	rp.ServeHTTP(r.hrw, r.req.HTTPRequest())
	if r.Status < http.StatusBadRequest {
		return nil
	}

	return errors.New(strings.ToLower(http.StatusText(r.Status)))
}

// Defer pushes the f onto the stack of functions that will be called after
// responding. Nil functions will be silently dropped.
func (r *Response) Defer(f func()) {
	if f != nil {
		r.deferredFuncs = append(r.deferredFuncs, f)
	}
}

// responseWriter used to tie the `Response` and the `http.ResponseWriter`
// together.
type responseWriter struct {
	sync.Mutex

	r  *Response
	w  http.ResponseWriter
	gw *gzip.Writer
}

// Header implements the `http.ResponseWriter`.
func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

// Write implements the `http.ResponseWriter`.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.r.Written {
		rw.WriteHeader(rw.r.Status)
	}

	var (
		n   int
		err error
	)

	if rw.gw != nil {
		n, err = rw.gw.Write(b)
	} else {
		n, err = rw.w.Write(b)
	}

	if err != nil {
		return 0, err
	}

	rw.r.ContentLength += int64(n)

	return n, nil
}

// WriteHeader implements the `http.ResponseWriter`.
func (rw *responseWriter) WriteHeader(status int) {
	rw.Lock()
	defer rw.Unlock()

	if rw.r.Written {
		return
	}

	if status == http.StatusOK && status != rw.r.Status {
		status = rw.r.Status
	}

	h := rw.w.Header()

	mt, _, _ := mime.ParseMediaType(h.Get("Content-Type"))
	if rw.r.Air.GzipEnabled &&
		stringSliceContains(rw.r.Air.GzipMIMETypes, mt) {
		if !strings.Contains(h.Get("Vary"), "Accept-Encoding") {
			h.Add("Vary", "Accept-Encoding")
		}

		if !rw.r.Gzipped && strings.Contains(
			rw.r.req.Header.Get("Accept-Encoding"),
			"gzip",
		) {
			if rw.gw, _ = gzip.NewWriterLevel(
				rw.w,
				rw.r.Air.GzipCompressionLevel,
			); rw.gw != nil {
				rw.r.Gzipped = true
				rw.r.Defer(func() {
					rw.gw.Close()
				})
			}
		}
	}

	if rw.r.Gzipped {
		h.Set("Content-Encoding", "gzip")
		h.Del("Content-Length")
	}

	if !rw.r.Air.DebugMode &&
		rw.r.Air.HTTPSEnforced &&
		rw.r.Air.server.server.TLSConfig != nil &&
		h.Get("Strict-Transport-Security") == "" {
		h.Set("Strict-Transport-Security", "max-age=31536000")
	}

	rw.w.WriteHeader(status)

	rw.r.Status = status
	rw.r.Written = true
}

// Flush implements the `http.Flusher`.
func (rw *responseWriter) Flush() {
	if rw.gw != nil {
		rw.gw.Flush()
	} else {
		rw.w.(http.Flusher).Flush()
	}
}

// Push implements the `http.Pusher`.
func (rw *responseWriter) Push(target string, pos *http.PushOptions) error {
	p, ok := rw.w.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}

	return p.Push(target, pos)
}

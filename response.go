package air

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
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
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// Response is an HTTP response.
//
// The `Response` not only represents HTTP/1.x responses, but also represents
// HTTP/2 responses, and always show as HTTP/2 responses.
type Response struct {
	// Air is where the response belongs.
	Air *Air

	// Status is the status code.
	//
	// See RFC 7231, section 6.
	//
	// For HTTP/1.x, it will be put in the Response-Line.
	//
	// For HTTP/2, it will be the ":status" pseudo-header.
	//
	// E.g.: 200
	Status int

	// Header is the header map.
	//
	// See RFC 7231, section 7.
	//
	// By setting the Trailer header to the names of the trailers which will
	// come later. In this case, those names of the header map are treated
	// as if they were trailers.
	//
	// The `Header` is basically the same for both HTTP/1.x and HTTP/2. The
	// only difference is that HTTP/2 requires header names to be lowercase
	// (for aesthetic reasons, this framework decided to follow this rule
	// implicitly, so please use the header name in HTTP/1.x style).
	//
	// E.g.: {"Foo": ["bar"]}
	Header http.Header

	// Body is the message body. It can be used to write a streaming
	// response.
	Body io.Writer

	// ContentLength records the length of the `Body`. The value -1
	// indicates that the length is unknown (it will continue to increase
	// as the data written to the `Body` increases). Values >= 0 indicate
	// that the given number of bytes has been written to the `Body`.
	ContentLength int64

	// Written indicates whether at least one byte has been written to the
	// client, or the connection has been hijacked.
	Written bool

	// Minified indicates whether the `Body` has been minified.
	Minified bool

	// Gzipped indicates whether the `Body` has been gzipped.
	Gzipped bool

	req               *Request
	hrw               http.ResponseWriter
	servingContent    bool
	serveContentError error
	deferredFuncs     []func()
}

// reset resets the r with the a, hrw and req.
func (r *Response) reset(a *Air, hrw http.ResponseWriter, req *Request) {
	r.Air = a
	r.Status = http.StatusOK
	r.ContentLength = -1
	r.Written = false
	r.Minified = false
	r.Gzipped = false
	r.req = req
	r.servingContent = false
	r.serveContentError = nil
	r.deferredFuncs = r.deferredFuncs[:0]

	rw := &responseWriter{
		r:   r,
		hrw: hrw,
	}

	hijacker, isHijacker := hrw.(http.Hijacker)
	pusher, isPusher := hrw.(http.Pusher)
	switch {
	case isHijacker && isPusher:
		r.SetHTTPResponseWriter(&struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
		}{
			rw,
			rw,
			&responseHijacker{
				r: r,
				h: hijacker,
			},
			pusher,
		})
	case isHijacker:
		r.SetHTTPResponseWriter(&struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
		}{
			rw,
			rw,
			&responseHijacker{
				r: r,
				h: hijacker,
			},
		})
	case isPusher:
		r.SetHTTPResponseWriter(&struct {
			http.ResponseWriter
			http.Flusher
			http.Pusher
		}{
			rw,
			rw,
			pusher,
		})
	default:
		r.SetHTTPResponseWriter(rw)
	}
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
// is that it handles range requests properly, sets the Content-Type response
// header, and handles the If-Match, If-Unmodified-Since, If-None-Match,
// If-Modified-Since and If-Range request headers.
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
		if err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}

		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		r.Header.Set("Content-Type", mimesniffer.Sniff(b[:n]))
	}

	if !r.Minified && r.Air.MinifierEnabled {
		mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if stringSliceContains(r.Air.MinifierMIMETypes, mt, true) {
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

// WriteHTML writes the h as a "text/html" content to the client.
func (r *Response) WriteHTML(h string) error {
	r.Header.Set("Content-Type", "text/html; charset=utf-8")
	return r.Write(strings.NewReader(h))
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

// WriteFile writes a file content targeted by the filename to the client.
func (r *Response) WriteFile(filename string) error {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return err
	} else if fi, err := os.Stat(filename); err != nil {
		return err
	} else if fi.IsDir() {
		p := r.req.RawPath()
		if !strings.HasSuffix(p, "/") {
			p = fmt.Sprint(path.Base(p), "/")
			if q := r.req.RawQuery(); q != "" {
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
				r.gzippable() {
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

// Redirect writes the url as a redirection to the client.
//
// The `Status` of the r will be the `http.StatusFound` if it is not a
// redirection status.
func (r *Response) Redirect(url string) error {
	if r.Written {
		return errors.New("air: request has been written")
	}

	if r.Status < http.StatusMultipleChoices ||
		r.Status >= http.StatusBadRequest {
		r.Status = http.StatusFound
	}

	http.Redirect(r.hrw, r.req.HTTPRequest(), url, r.Status)

	return nil
}

// Flush flushes any buffered data to the client.
//
// The `Flush` does nothing if it is not supported by the underlying
// `http.ResponseWriter` of the r.
func (r *Response) Flush() {
	if flusher, ok := r.hrw.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Push initiates an HTTP/2 server push. This constructs a synthetic request
// using the target and pos, serializes that request into a "PUSH_PROMISE"
// frame, then dispatches that request using the server's request handler. If
// pos is nil, default options are used.
//
// The target must either be an absolute path (like "/path") or an absolute URL
// that contains a valid authority and the same scheme as the parent request. If
// the target is a path, it will inherit the scheme and authority of the parent
// request.
//
// The `Push` returns `http.ErrNotSupported` if the client has disabled it or if
// it is not supported by the underlying `http.ResponseWriter` of the r.
func (r *Response) Push(target string, pos *http.PushOptions) error {
	if pusher, ok := r.hrw.(http.Pusher); ok {
		return pusher.Push(target, pos)
	}

	return http.ErrNotSupported
}

// WebSocket switches the connection of the r to the WebSocket protocol. See RFC
// 6455.
func (r *Response) WebSocket() (*WebSocket, error) {
	if r.Written {
		return nil, errors.New("air: request has been written")
	}

	r.Status = http.StatusSwitchingProtocols

	conn, err := (&websocket.Upgrader{
		HandshakeTimeout: r.Air.WebSocketHandshakeTimeout,
		Subprotocols:     r.Air.WebSocketSubprotocols,
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
	}).Upgrade(r.hrw, r.req.HTTPRequest(), r.Header)
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
		if errors.Is(err, websocket.ErrCloseSent) {
			return nil
		}

		var ne net.Error
		if errors.As(err, &ne) && ne.Temporary() {
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

// ProxyPass passes the request to the target and writes the response from the
// target to the client by using the reverse proxy technique. If the rp is nil,
// the default instance of the `ReverseProxy` will be used.
//
// The target must be based on the HTTP protocol (such as HTTP, WebSocket and
// gRPC). So, the scheme of the target must be "http", "https", "ws", "wss",
// "grpc" or "grpcs".
func (r *Response) ProxyPass(target string, rp *ReverseProxy) error {
	if r.Written {
		return errors.New("air: request has been written")
	}

	if rp == nil {
		rp = &ReverseProxy{}
	}

	targetMethod := r.req.Method
	if mrm := rp.ModifyRequestMethod; mrm != nil {
		m, err := mrm(targetMethod)
		if err != nil {
			return err
		}

		targetMethod = m
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

	reqPath := r.req.Path
	if mrp := rp.ModifyRequestPath; mrp != nil {
		p, err := mrp(reqPath)
		if err != nil {
			return err
		}

		reqPath = p
	}

	if reqPath == "" {
		reqPath = "/"
	}

	reqURL, err := url.ParseRequestURI(reqPath)
	if err != nil {
		return err
	}

	targetURL.Path = path.Join(targetURL.Path, reqURL.Path)
	targetURL.RawPath = path.Join(targetURL.RawPath, reqURL.RawPath)
	if targetURL.RawQuery == "" || reqURL.RawQuery == "" {
		targetURL.RawQuery = fmt.Sprint(
			targetURL.RawQuery,
			reqURL.RawQuery,
		)
	} else {
		targetURL.RawQuery = fmt.Sprint(
			targetURL.RawQuery,
			"&",
			reqURL.RawQuery,
		)
	}

	targetHeader := r.req.Header.Clone()
	if mrh := rp.ModifyRequestHeader; mrh != nil {
		h, err := mrh(targetHeader)
		if err != nil {
			return err
		}

		targetHeader = h
	}

	if _, ok := targetHeader["User-Agent"]; !ok {
		// Explicitly disable the User-Agent header so it's not set to
		// default value.
		targetHeader.Set("User-Agent", "")
	}

	targetBody := r.req.Body
	if mrb := rp.ModifyRequestBody; mrb != nil {
		b, err := mrb(targetBody)
		if err != nil {
			return err
		}

		targetBody = b
	}

	var reverseProxyError error
	hrp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Method = targetMethod
			req.URL = targetURL
			req.Header = targetHeader
			req.Body = targetBody

			// TODO: Remove the following line when the
			// "net/http/httputil" of the minimum supported Go
			// version of Air has fixed this bug.
			req.Host = ""
		},
		FlushInterval: 100 * time.Millisecond,
		Transport:     r.Air.reverseProxyTransport,
		ErrorLog:      r.Air.ErrorLogger,
		BufferPool:    r.Air.reverseProxyBufferPool,
		ModifyResponse: func(res *http.Response) error {
			if mrs := rp.ModifyResponseStatus; mrs != nil {
				s, err := mrs(res.StatusCode)
				if err != nil {
					return err
				}

				res.StatusCode = s
			}

			if mrh := rp.ModifyResponseHeader; mrh != nil {
				h, err := mrh(res.Header)
				if err != nil {
					return err
				}

				res.Header = h
			}

			if mrb := rp.ModifyResponseBody; mrb != nil {
				b, err := mrb(res.Body)
				if err != nil {
					return err
				}

				res.Body = b
			}

			r.Gzipped = httpguts.HeaderValuesContainsToken(
				res.Header["Content-Encoding"],
				"gzip",
			)

			return nil
		},
		ErrorHandler: func(
			_ http.ResponseWriter,
			_ *http.Request,
			err error,
		) {
			if r.Status < http.StatusBadRequest {
				r.Status = http.StatusBadGateway
			}

			reverseProxyError = err
		},
	}
	switch targetURL.Scheme {
	case "grpc", "grpcs":
		hrp.FlushInterval /= 100 // For gRPC streaming
	}

	if rp.Transport != nil {
		hrp.Transport = rp.Transport
	}

	defer func() {
		r := recover()
		if r == nil || r == http.ErrAbortHandler {
			return
		}

		panic(r)
	}()

	switch targetURL.Scheme {
	case "ws", "wss":
		r.Status = http.StatusSwitchingProtocols
	}

	hrp.ServeHTTP(r.hrw, r.req.HTTPRequest())

	return reverseProxyError
}

// Defer pushes the f onto the stack of functions that will be called after
// responding. Nil functions will be silently dropped.
func (r *Response) Defer(f func()) {
	if f != nil {
		r.deferredFuncs = append(r.deferredFuncs, f)
	}
}

// gzippable reports whether the r is gzippable.
func (r *Response) gzippable() bool {
	for _, ae := range strings.Split(
		strings.Join(r.req.Header["Accept-Encoding"], ","),
		",",
	) {
		ae = strings.TrimSpace(ae)
		ae = strings.Split(ae, ";")[0]
		ae = strings.ToLower(ae)
		if ae == "gzip" {
			return true
		}
	}

	return false
}

// ReverseProxy is used by the `Response.ProxyPass` to achieve the reverse proxy
// technique.
type ReverseProxy struct {
	// Transport is used to perform the request to the target.
	//
	// Normally the `Transport` should be nil, which means that a default
	// and well-improved one will be used. If the `Transport` is not nil, it
	// is responsible for keeping the `Response.ProxyPass` working properly.
	Transport http.RoundTripper

	// ModifyRequestMethod modifies the method of the request to the target.
	ModifyRequestMethod func(method string) (string, error)

	// ModifyRequestPath modifies the path of the request to the target.
	//
	// Note that the path contains the query part. Therefore, the returned
	// path must also be in this format.
	ModifyRequestPath func(path string) (string, error)

	// ModifyRequestHeader modifies the header of the request to the target.
	ModifyRequestHeader func(header http.Header) (http.Header, error)

	// ModifyRequestBody modifies the body of the request from the target.
	//
	// It is the caller's responsibility to close the returned
	// `io.ReadCloser`, which means that the `Response.ProxyPass` will be
	// responsible for closing it.
	ModifyRequestBody func(body io.ReadCloser) (io.ReadCloser, error)

	// ModifyResponseStatus modifies the status of the response from the
	// target.
	ModifyResponseStatus func(status int) (int, error)

	// ModifyResponseHeader modifies the header of the response from the
	// target.
	ModifyResponseHeader func(header http.Header) (http.Header, error)

	// ModifyResponseBody modifies the body of the response from the target.
	//
	// It is the caller's responsibility to close the returned
	// `io.ReadCloser`, which means that the `Response.ProxyPass` will be
	// responsible for closing it.
	ModifyResponseBody func(body io.ReadCloser) (io.ReadCloser, error)
}

// responseWriter is used to tie the `Response` and `http.ResponseWriter`
// together.
type responseWriter struct {
	sync.Mutex

	r   *Response
	hrw http.ResponseWriter
	cw  *countWriter
	gw  *gzip.Writer
}

// Header implements the `http.ResponseWriter`.
func (rw *responseWriter) Header() http.Header {
	return rw.hrw.Header()
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

	rw.cw = &countWriter{
		w: rw.hrw,
		c: &rw.r.ContentLength,
	}

	rw.handleGzip()
	rw.hrw.WriteHeader(status)

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

	w := io.Writer(rw.cw)
	if rw.gw != nil {
		w = rw.gw
	}

	return w.Write(b)
}

// Flush implements the `http.Flusher`.
func (rw *responseWriter) Flush() {
	if rw.gw != nil {
		rw.gw.Flush()
	}

	if flusher, ok := rw.hrw.(http.Flusher); ok {
		flusher.Flush()
	}
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
		); !stringSliceContains(rw.r.Air.GzipMIMETypes, mt, true) {
			return
		}

		if rw.r.gzippable() {
			rw.gw, _ = rw.r.Air.gzipWriterPool.Get().(*gzip.Writer)
			if rw.gw == nil {
				return
			}

			rw.gw.Reset(rw.cw)
			rw.r.Defer(func() {
				if rw.r.ContentLength == 0 {
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

		// See RFC 7232, section 2.3.3.
		if et := rw.r.Header.Get("ETag"); et != "" {
			et = strings.TrimSuffix(et, `"`)
			et = fmt.Sprint(et, `-gzip"`)
			rw.r.Header.Set("ETag", et)
		}
	}

	if !httpguts.HeaderValuesContainsToken(
		rw.r.Header["Vary"],
		"Accept-Encoding",
	) {
		rw.r.Header.Add("Vary", "Accept-Encoding")
	}
}

// responseHijacker is used to tie the `Response` and `http.Hijacker` together.
type responseHijacker struct {
	r *Response
	h http.Hijacker
}

// Hijack implements the `http.Hijacker`.
func (rh *responseHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, rw, err := rh.h.Hijack()
	if err == nil {
		rh.r.Written = true
	}

	return c, rw, err
}

// countWriter is used to count the number of bytes written to the underlying
// `io.Writer`.
type countWriter struct {
	w io.Writer
	c *int64
}

// Write implements the `io.Writer`.
func (cw *countWriter) Write(b []byte) (int, error) {
	n, err := cw.w.Write(b)
	*cw.c += int64(n)
	return n, err
}

// reverseProxyTransport is a transport with the reverse proxy support.
type reverseProxyTransport struct {
	hTransport   *http.Transport
	h2Transport  *http2.Transport
	h2cTransport *http2.Transport
}

// newReverseProxyTransport returns a new instance of the
// `reverseProxyTransport`.
func newReverseProxyTransport() *reverseProxyTransport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	return &reverseProxyTransport{
		hTransport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			DisableCompression:    true,
			MaxIdleConnsPerHost:   200,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
		h2Transport: &http2.Transport{
			DialTLS: func(
				network string,
				address string,
				tlsConfig *tls.Config,
			) (net.Conn, error) {
				return tls.DialWithDialer(
					dialer,
					network,
					address,
					tlsConfig,
				)
			},
			DisableCompression: true,
		},
		h2cTransport: &http2.Transport{
			DialTLS: func(
				network string,
				address string,
				_ *tls.Config,
			) (net.Conn, error) {
				return dialer.Dial(network, address)
			},
			DisableCompression: true,
			AllowHTTP:          true,
		},
	}
}

// RoundTrip implements the `http.RoundTripper`.
func (rpt *reverseProxyTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	var transport http.RoundTripper
	switch req.URL.Scheme {
	case "ws":
		req.URL.Scheme = "http"
		transport = rpt.hTransport
	case "wss":
		req.URL.Scheme = "https"
		transport = rpt.hTransport
	case "grpc":
		req.URL.Scheme = "http"
		transport = rpt.h2cTransport
	case "grpcs":
		req.URL.Scheme = "https"
		transport = rpt.h2Transport
	default:
		transport = rpt.hTransport
	}

	return transport.RoundTrip(req)
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

package air

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
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
		for k, v := range r.Headers {
			r.writer.Header().Set(k, v)
		}
		for _, c := range r.Cookies {
			if v := c.String(); v != "" {
				r.writer.Header().Add("Set-Cookie", v)
			}
		}
		r.writer.WriteHeader(r.StatusCode)
		r.Written = true
	}
	n, err := r.writer.Write(b)
	r.Size += n
	return err
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
	if b, err = minifierSingleton.minify(contentType, b); err != nil {
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
		if err := rendererSingleton.render(buf, t, m); err != nil {
			return err
		}
	}
	return r.HTML(buf.String())
}

// Stream responds to the client with the contentType streaming content reader.
func (r *Response) Stream(contentType string, reader io.Reader) error {
	if err := r.Blob(contentType, nil); err != nil {
		return err
	}
	_, err := io.Copy(r.writer, reader)
	return err
}

// File responds to the client with a file content with the file.
func (r *Response) File(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}

	for k, v := range r.Headers {
		r.writer.Header().Set(k, v)
	}

	for _, c := range r.Cookies {
		if v := c.String(); v != "" {
			r.writer.Header().Add("Set-Cookie", v)
		}
	}

	if a, err := cofferSingleton.asset(file); err != nil {
		return err
	} else if a != nil {
		http.ServeContent(
			r.writer,
			r.request.request,
			a.name,
			a.modTime,
			a.reader,
		)
	} else {
		http.ServeFile(r.writer, r.request.request, file)
	}

	r.Written = true

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

package air

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

// Response represents the current HTTP response.
//
// It's embedded with the `http.ResponseWriter`.
type Response struct {
	http.ResponseWriter

	context *Context

	statusCode int
	size       int
	written    bool

	Data JSONMap
}

// NewResponse returns a pointer of a new instance of the `Response`.
func NewResponse(c *Context) *Response {
	return &Response{
		context: c,
		Data:    make(JSONMap),
	}
}

// Write implements the `http.ResponseWriter#Write()`.
func (res *Response) Write(b []byte) (int, error) {
	if !res.written {
		res.WriteHeader(http.StatusOK)
	}
	n, err := res.ResponseWriter.Write(b)
	res.size += n
	return n, err
}

// WriteHeader implements the `http.ResponseWriter#WriteHeader()`.
func (res *Response) WriteHeader(statusCode int) {
	if res.written {
		res.context.Air.Logger.Warn("response already written")
		return
	}
	res.statusCode = statusCode
	res.ResponseWriter.WriteHeader(statusCode)
	res.written = true
}

// StatusCode returns the HTTP status code of the res.
func (res *Response) StatusCode() int {
	return res.statusCode
}

// Size returns the number of bytes already written into the HTTP body of the res.
func (res *Response) Size() int {
	return res.size
}

// Written reports whether the HTTP body of the res is already written.
func (res *Response) Written() bool {
	return res.written
}

// SetCookie adds a "Set-Cookie" header in the res. The provided cookie must have a valid `Name`.
// Invalid cookies may be silently dropped.
func (res *Response) SetCookie(cookie *http.Cookie) {
	http.SetCookie(res.ResponseWriter, cookie)
}

// Hijack lets the caller take over the connection. After a call to this method, the HTTP server
// will not do anything else with the connection.
func (res *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return res.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify returns a channel that receives at most a single value (true) when the client
// connection has gone away.
func (res *Response) CloseNotify() <-chan bool {
	return res.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush sends any buffered data to the client.
func (res *Response) Flush() {
	res.ResponseWriter.(http.Flusher).Flush()
}

// Push initiates an HTTP/2 server push with the target and an optional pos.
func (res *Response) Push(target string, pos *http.PushOptions) error {
	p, pok := res.ResponseWriter.(http.Pusher)
	if !pok {
		return ErrDisabledHTTP2
	}
	return p.Push(target, pos)
}

// Render renders one or more HTML templates with the `Data` of the res and sends a "text/html" HTTP
// response. The default `Renderer` does it by using the `template.Template`. The results rendered
// by the former can be inherited by accessing the `Data["InheritedHTML"]` of the res.
func (res *Response) Render(templates ...string) error {
	buf := &bytes.Buffer{}
	for _, t := range templates {
		res.Data["InheritedHTML"] = template.HTML(buf.String())
		buf.Reset()
		err := res.context.Air.Renderer.Render(buf, t, res.Data)
		if err != nil {
			return err
		}
	}
	return res.Blob(MIMETextHTML, buf.Bytes())
}

// HTML sends a "text/html" HTTP response with the html.
func (res *Response) HTML(html string) error {
	return res.Blob(MIMETextHTML, []byte(html))
}

// String sends a "text/plain" HTTP response with the s.
func (res *Response) String(s string) error {
	return res.Blob(MIMETextPlain, []byte(s))
}

// JSON sends an "application/json" HTTP response with the type i.
func (res *Response) JSON(i interface{}) error {
	b, err := json.Marshal(i)
	if res.context.Air.Config.DebugMode {
		b, err = json.MarshalIndent(i, "", "\t")
	}
	if err != nil {
		return err
	}
	return res.Blob(MIMEApplicationJSON, b)
}

// JSONP sends an "application/javascript" HTTP response with the type i. It uses the callback to
// construct the JSONP payload.
func (res *Response) JSONP(i interface{}, callback string) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	b = append([]byte(callback+"("), b...)
	b = append(b, []byte(");")...)
	return res.Blob(MIMEApplicationJavaScript, b)
}

// XML sends an "application/xml" HTTP response with the type i.
func (res *Response) XML(i interface{}) error {
	b, err := xml.Marshal(i)
	if res.context.Air.Config.DebugMode {
		b, err = xml.MarshalIndent(i, "", "\t")
	}
	if err != nil {
		return err
	}
	return res.Blob(MIMEApplicationXML, append([]byte(xml.Header), b...))
}

// YAML sends an "application/x-yaml" HTTP response with the type i.
func (res *Response) YAML(i interface{}) error {
	b, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	return res.Blob(MIMEApplicationYAML, b)
}

// Blob sends a blob HTTP response with the contentType and the b.
func (res *Response) Blob(contentType string, b []byte) error {
	res.Header().Set(HeaderContentType, contentType)
	_, err := res.Write(b)
	return err
}

// Stream sends a streaming HTTP response with the contentType and the r.
func (res *Response) Stream(contentType string, r io.Reader) error {
	res.Header().Set(HeaderContentType, contentType)
	_, err := io.Copy(res, r)
	return err
}

// File sends a file HTTP response with the file.
func (res *Response) File(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return ErrNotFound
	}
	defer f.Close()

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.Join(file, "index.html")
		f, err = os.Open(file)
		if err != nil {
			return ErrNotFound
		}
		defer f.Close()
		if fi, err = f.Stat(); err != nil {
			return err
		}
	}

	http.ServeContent(res, res.context.Request.Request, fi.Name(), fi.ModTime(), f)

	return nil
}

// Attachment sends an HTTP response with the file and the filename as attachment, prompting client
// to save the file.
func (res *Response) Attachment(file, filename string) error {
	return res.contentDisposition("attachment", file, filename)
}

// Inline sends an HTTP response with the file and the filename as inline, opening the file in the
// browser.
func (res *Response) Inline(file, filename string) error {
	return res.contentDisposition("inline", file, filename)
}

// contentDisposition sends an HTTP response with the file and the filename as the dispositionType.
func (res *Response) contentDisposition(dispositionType, file, filename string) error {
	res.Header().Set(HeaderContentDisposition, fmt.Sprintf("%s; filename=%s",
		dispositionType, filename))
	return res.File(file)
}

// NoContent sends an HTTP response with no body.
func (res *Response) NoContent() error { return nil }

// Redirect redirects the current HTTP request to the url with the statusCode.
func (res *Response) Redirect(statusCode int, url string) error {
	if statusCode < http.StatusMultipleChoices || statusCode > http.StatusTemporaryRedirect {
		return ErrInvalidRedirectCode
	}
	res.Header().Set(HeaderLocation, url)
	res.WriteHeader(statusCode)
	return nil
}

// reset resets all fields in the res.
func (res *Response) reset() {
	res.ResponseWriter = nil
	res.statusCode = 0
	res.size = 0
	res.written = false
	for k := range res.Data {
		delete(res.Data, k)
	}
}

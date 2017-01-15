package air

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

// Response represents the current HTTP response.
//
// It's embedded with `http.ResponseWriter`.
type Response struct {
	http.ResponseWriter

	statusCode int
	size       int
	written    bool

	context *Context

	Data JSONMap
}

// newResponse returns a pointer of a new instance of `Response`.
func newResponse(c *Context) *Response {
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

// Written returns whether the HTTP body of the res is already written.
func (res *Response) Written() bool {
	return res.written
}

// SetCookie adds a "Set-Cookie" header in the res. The provided cookie must have a valid `Name`.
// Invalid cookies may be silently dropped.
func (res *Response) SetCookie(cookie *http.Cookie) {
	http.SetCookie(res.ResponseWriter, cookie)
}

// Hijack lets the caller take over the connection. After a call to this method, the `server` will
// not do anything else with the connection.
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
		return errors.New("the HTTP/2 has been disabled")
	}
	return p.Push(target, pos)
}

// Render renders a template with the `Data` and `Data["template"]` or `Data["templates"]` of the
// res and sends a "text/html" response.
func (res *Response) Render() error {
	t, tok := res.Data["template"].(string)
	ts, tsok := res.Data["templates"].([]string)
	if !tok && !tsok {
		return errors.New("both Data[\"template\"] and Data[\"templates\"] are not setted")
	}

	buf := &bytes.Buffer{}
	if tok {
		err := res.context.Air.Renderer.Render(buf, t, res.Data)
		if err != nil {
			return err
		}
	} else {
		for _, t := range ts {
			res.Data["InheritedHTML"] = template.HTML(buf.String())
			buf.Reset()
			err := res.context.Air.Renderer.Render(buf, t, res.Data)
			if err != nil {
				return err
			}
		}
	}

	return res.Blob(MIMETextHTML, buf.Bytes())
}

// HTML sends an HTTP response with the `Data["html"]` of the res.
func (res *Response) HTML() error {
	h, ok := res.Data["html"].(string)
	if !ok {
		return errors.New("Data[\"html\"] is not setted")
	}
	return res.Blob(MIMETextHTML, []byte(h))
}

// String sends a string response with the `Data["string"]` of the res.
func (res *Response) String() error {
	s, ok := res.Data["string"].(string)
	if !ok {
		return errors.New("Data[\"string\"] is not setted")
	}
	return res.Blob(MIMETextPlain, []byte(s))
}

// JSON sends a JSON response with the `Data["json"]` of the res.
func (res *Response) JSON() error {
	j, ok := res.Data["json"]
	if !ok {
		return errors.New("Data[\"json\"] is not setted")
	}

	b, err := json.Marshal(j)
	if res.context.Air.Config.DebugMode {
		b, err = json.MarshalIndent(j, "", "\t")
	}
	if err != nil {
		return err
	}

	return res.Blob(MIMEApplicationJSON, b)
}

// JSONP sends a JSONP response with the `Data["jsonp"]` of the res. It uses the `Data["callback"]`
// of the res to construct the JSONP payload.
func (res *Response) JSONP() error {
	j, ok := res.Data["jsonp"]
	cb, _ := res.Data["callback"].(string)
	if !ok {
		return errors.New("Data[\"jsonp\"] is not setted")
	}

	b, err := json.Marshal(j)
	if err != nil {
		return err
	}

	b = append([]byte(cb+"("), b...)
	b = append(b, []byte(");")...)

	return res.Blob(MIMEApplicationJavaScript, b)
}

// XML sends an XML response with the `Data["xml"]` of the res.
func (res *Response) XML() error {
	x, ok := res.Data["xml"]
	if !ok {
		return errors.New("Data[\"xml\"] is not setted")
	}

	b, err := xml.Marshal(x)
	if res.context.Air.Config.DebugMode {
		b, err = xml.MarshalIndent(x, "", "\t")
	}
	if err != nil {
		return err
	}

	return res.Blob(MIMEApplicationXML, append([]byte(xml.Header), b...))
}

// Blob sends a blob response with the contentType and the b.
func (res *Response) Blob(contentType string, b []byte) error {
	res.Header().Set(HeaderContentType, contentType)
	_, err := res.Write(b)
	return err
}

// Stream sends a streaming response with the contentType and the r.
func (res *Response) Stream(contentType string, r io.Reader) error {
	res.Header().Set(HeaderContentType, contentType)
	_, err := io.Copy(res, r)
	return err
}

// File sends a response with the `Data["file"]` of the res.
func (res *Response) File() error {
	file, ok := res.Data["file"].(string)
	if !ok {
		return errors.New("Data[\"file\"] is not setted")
	}

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

// Attachment sends a response with the `Data["file"]` and the `Data["filename"]` of the res as
// attachment, prompting client to save the file.
func (res *Response) Attachment() error {
	return res.contentDisposition("attachment")
}

// Inline sends a response with the `Data["file"]` and the `Data["filename"]` of the res as inline,
// opening the file in the browser.
func (res *Response) Inline() error {
	return res.contentDisposition("inline")
}

// contentDisposition sends a response with the `Data["file"]` and the `Data["filename"]` as the
// dispositionType.
func (res *Response) contentDisposition(dispositionType string) error {
	fn, ok := res.Data["filename"].(string)
	if !ok {
		return errors.New("Data[\"filename\"] is not setted")
	}
	res.Header().Set(HeaderContentDisposition, fmt.Sprintf("%s; filename=%s",
		dispositionType, fn))
	return res.File()
}

// NoContent sends a response with no body.
func (res *Response) NoContent() error { return nil }

// Redirect redirects the request to the url with the statusCode.
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

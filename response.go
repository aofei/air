package air

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Response represents the current HTTP response.
//
// It's embedded with the `http.ResponseWriter`, the `http.Hijacker`, the
// `http.CloseNotifier`, the `http.Flusher` and the `http.Pusher`.
type Response struct {
	http.ResponseWriter
	http.Hijacker
	http.CloseNotifier
	http.Flusher
	http.Pusher

	context *Context

	StatusCode int
	Size       int
	Written    bool
	Data       Map
}

// NewResponse returns a pointer of a new instance of the `Response`.
func NewResponse(c *Context) *Response {
	return &Response{
		context:    c,
		StatusCode: http.StatusOK,
		Data:       Map{},
	}
}

// Write implements the `http.ResponseWriter#Write()`.
func (r *Response) Write(b []byte) (int, error) {
	if !r.Written {
		r.WriteHeader(r.StatusCode)
	}
	n, err := r.ResponseWriter.Write(b)
	r.Size += n
	return n, err
}

// WriteHeader implements the `http.ResponseWriter#WriteHeader()`.
func (r *Response) WriteHeader(statusCode int) {
	if r.Written {
		r.context.Air.Logger.Warn("response already written")
		return
	}
	r.ResponseWriter.WriteHeader(statusCode)
	r.StatusCode = statusCode
	r.Written = true
}

// SetCookie adds a "Set-Cookie" header in the r. The provided cookie must have
// a valid `Name`. Invalid cookies may be silently dropped.
func (r *Response) SetCookie(cookie *http.Cookie) {
	http.SetCookie(r, cookie)
}

// NoContent sends an HTTP response with the statusCode and no body.
func (r *Response) NoContent(statusCode int) error {
	r.WriteHeader(statusCode)
	return nil
}

// Redirect redirects the current HTTP request to the url with the statusCode.
func (r *Response) Redirect(statusCode int, url string) error {
	r.Header().Set("Location", url)
	return r.NoContent(statusCode)
}

// Blob sends a blob HTTP response with the contentType and the b.
func (r *Response) Blob(contentType string, b []byte) error {
	r.Header().Set("Content-Type", contentType)
	_, err := r.Write(b)
	return err
}

// String sends a "text/plain" HTTP response with the s.
func (r *Response) String(s string) error {
	return r.Blob("text/plain; charset=utf-8", []byte(s))
}

// JSON sends an "application/json" HTTP response with the type i.
func (r *Response) JSON(i interface{}) error {
	b, err := json.Marshal(i)
	if r.context.Air.Config.DebugMode {
		b, err = json.MarshalIndent(i, "", "\t")
	}
	if err != nil {
		return err
	}
	if r.context.Air.Config.MinifierEnabled {
		b, err = r.context.Air.Minifier.Minify("application/json", b)
		if err != nil {
			return err
		}
	}
	return r.Blob("application/json; charset=utf-8", b)
}

// XML sends an "application/xml" HTTP response with the type i.
func (r *Response) XML(i interface{}) error {
	b, err := xml.Marshal(i)
	if r.context.Air.Config.DebugMode {
		b, err = xml.MarshalIndent(i, "", "\t")
	}
	if err != nil {
		return err
	}
	b = append([]byte(xml.Header), b...)
	if r.context.Air.Config.MinifierEnabled {
		b, err = r.context.Air.Minifier.Minify("text/xml", b)
		if err != nil {
			return err
		}
	}
	return r.Blob("application/xml; charset=utf-8", b)
}

// HTML sends a "text/html" HTTP response with the html.
func (r *Response) HTML(html string) error {
	b := []byte(html)
	if r.context.Air.Config.MinifierEnabled {
		var err error
		b, err = r.context.Air.Minifier.Minify("text/html", b)
		if err != nil {
			return err
		}
	}
	return r.Blob("text/html; charset=utf-8", b)
}

// Render renders one or more HTML templates with the `Data` of the r and sends
// a "text/html" HTTP response. The default `Renderer` does it by using the
// `template.Template`. the results rendered by the former can be inherited by
// accessing the `Data["InheritedHTML"]` of the r.
func (r *Response) Render(templates ...string) error {
	buf := &bytes.Buffer{}
	for _, t := range templates {
		r.Data["InheritedHTML"] = template.HTML(buf.String())
		buf.Reset()
		err := r.context.Air.Renderer.Render(buf, t, r.Data)
		if err != nil {
			return err
		}
	}
	return r.HTML(buf.String())
}

// Stream sends a streaming HTTP response with the contentType and the reader.
func (r *Response) Stream(contentType string, reader io.Reader) error {
	r.Header().Set("Content-Type", contentType)
	_, err := io.Copy(r, reader)
	return err
}

// File sends a file HTTP response with the file.
func (r *Response) File(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}

	abs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	if a := r.context.Air.Coffer.Asset(abs); a != nil {
		http.ServeContent(
			r,
			r.context.Request.Request,
			a.Name(),
			a.ModTime(),
			a,
		)
	} else {
		http.ServeFile(r, r.context.Request.Request, abs)
	}

	return nil
}

// Attachment sends an HTTP response with the file and the filename as
// attachment, prompting client to save the file.
func (r *Response) Attachment(file, filename string) error {
	r.Header().Set("Content-Disposition", "attachment; filename="+filename)
	return r.File(file)
}

// Inline sends an HTTP response with the file and the filename as inline,
// opening the file in the browser.
func (r *Response) Inline(file, filename string) error {
	r.Header().Set("Content-Disposition", "inline; filename="+filename)
	return r.File(file)
}

// feed feeds the rw into where it should be.
func (r *Response) feed(rw http.ResponseWriter) {
	r.ResponseWriter = rw
	r.Hijacker, _ = rw.(http.Hijacker)
	r.CloseNotifier, _ = rw.(http.CloseNotifier)
	r.Flusher, _ = rw.(http.Flusher)
	r.Pusher, _ = rw.(http.Pusher)
}

// reset resets all fields in the r.
func (r *Response) reset() {
	r.ResponseWriter = nil
	r.StatusCode = http.StatusOK
	r.Size = 0
	r.Written = false
	for k := range r.Data {
		delete(r.Data, k)
	}
}

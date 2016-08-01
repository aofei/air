package air

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
)

// Context represents the context of the current HTTP request. It holds request and
// response objects, path, path parameters, data and registered handler.
type Context struct {
	goContext   context.Context
	Request     *Request
	Response    *Response
	Path        string
	ParamNames  []string
	ParamValues []string
	Data        map[string]interface{}
	StatusCode  int
	Handler     HandlerFunc
	Air         *Air
}

// Deadline returns the time when work done on behalf of this context
// should be canceled.  Deadline returns ok==false when no deadline is
// set.  Successive calls to Deadline return the same results.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.goContext.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.  Done may return nil if this context can
// never be canceled.  Successive calls to Done return the same value.
func (c *Context) Done() <-chan struct{} {
	return c.goContext.Done()
}

// Err returns a non-nil error value after Done is closed.  Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed.  No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (c *Context) Err() error {
	return c.goContext.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key.  Successive calls to Value with
// the same key returns the same result.
func (c *Context) Value(key interface{}) interface{} {
	return c.goContext.Value(key)
}

// P returns path parameter by index.
func (c *Context) P(i int) (value string) {
	l := len(c.ParamNames)
	if i < l {
		value = c.ParamValues[i]
	}
	return
}

// Param returns path parameter by name.
func (c *Context) Param(name string) (value string) {
	l := len(c.ParamNames)
	for i, n := range c.ParamNames {
		if n == name && i < l {
			value = c.ParamValues[i]
			break
		}
	}
	return
}

// QueryParam returns the query param for the provided name. It is an alias
// for `URI#QueryParam()`.
func (c *Context) QueryParam(name string) string {
	return c.Request.URI.QueryParam(name)
}

// QueryParams returns the query parameters as map.
// It is an alias for `URI#QueryParams()`.
func (c *Context) QueryParams() map[string][]string {
	return c.Request.URI.QueryParams()
}

// FormValue returns the form field value for the provided name. It is an
// alias for `Request#FormValue()`.
func (c *Context) FormValue(name string) string {
	return c.Request.FormValue(name)
}

// FormParams returns the form parameters as map.
// It is an alias for `Request#FormParams()`.
func (c *Context) FormParams() map[string][]string {
	return c.Request.FormParams()
}

// FormFile returns the multipart form file for the provided name. It is an
// alias for `Request#FormFile()`.
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	return c.Request.FormFile(name)
}

// MultipartForm returns the multipart form.
// It is an alias for `Request#MultipartForm()`.
func (c *Context) MultipartForm() (*multipart.Form, error) {
	return c.Request.MultipartForm()
}

// Cookie returns the named cookie provided in the request.
// It is an alias for `Request#Cookie()`.
func (c *Context) Cookie(name string) (Cookie, error) {
	return c.Request.Cookie(name)
}

// SetCookie adds a `Set-Cookie` header in HTTP response.
// It is an alias for `Response#SetCookie()`.
func (c *Context) SetCookie(cookie Cookie) {
	c.Response.SetCookie(cookie)
}

// Cookies returns the HTTP cookies sent with the request.
// It is an alias for `Request#Cookies()`.
func (c *Context) Cookies() []Cookie {
	return c.Request.Cookies()
}

// Set saves data in the context.
func (c *Context) Set(key string, val interface{}) {
	c.goContext = context.WithValue(c.goContext, key, val)
}

// Get retrieves data from the context.
func (c *Context) Get(key string) interface{} {
	return c.goContext.Value(key)
}

// Bind binds the request body into provided type `i`. The default binder
// does it based on Content-Type header.
func (c *Context) Bind(i interface{}) error {
	return c.Air.Binder.Bind(i, c)
}

// Render renders a template with `Context#Data` and `Context#Data["template"]`
// and sends a text/html response with `Context#StatusCode`.
func (c *Context) Render() (err error) {
	if c.Air.Renderer == nil {
		return ErrRendererNotRegistered
	}
	buf := new(bytes.Buffer)
	if err = c.Air.Renderer.Render(buf, c.Data["template"].(string), c); err != nil {
		return
	}
	c.Response.Header.Set(HeaderContentType, MIMETextHTML)
	c.Response.WriteHeader(c.StatusCode)
	_, err = c.Response.Write(buf.Bytes())
	return
}

// HTML sends an HTTP response with `Context#StatusCode` and `Context#Data["html"]`.
func (c *Context) HTML() (err error) {
	data := c.Data["html"].(string)
	c.Response.Header.Set(HeaderContentType, MIMETextHTML)
	c.Response.WriteHeader(c.StatusCode)
	_, err = c.Response.Write([]byte(data))
	return
}

// String sends a string response with `Context#StatusCode` and `Context#Data["string"]`.
func (c *Context) String() (err error) {
	data := c.Data["string"].(string)
	c.Response.Header.Set(HeaderContentType, MIMETextPlain)
	c.Response.WriteHeader(c.StatusCode)
	_, err = c.Response.Write([]byte(data))
	return
}

// JSON sends a JSON response with `Context#StatusCode` and `Context#Data["json"]`.
func (c *Context) JSON() (err error) {
	data := c.Data["json"]
	b, err := json.Marshal(data)
	if c.Air.Debug {
		b, err = json.MarshalIndent(data, "", "\t")
	}
	if err != nil {
		return err
	}
	return c.JSONBlob(b)
}

// JSONBlob sends a JSON blob response with `Context#StatusCode`.
func (c *Context) JSONBlob(b []byte) (err error) {
	c.Response.Header.Set(HeaderContentType, MIMEApplicationJSON)
	c.Response.WriteHeader(c.StatusCode)
	_, err = c.Response.Write(b)
	return
}

// JSONP sends a JSONP response with `Context#StatusCode` and `Context#Data["jsonp"]`.
// It uses `Context#Data["callback"]` to construct the JSONP payload.
func (c *Context) JSONP() (err error) {
	data := c.Data["jsonp"]
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.Response.Header.Set(HeaderContentType, MIMEApplicationJavaScript)
	c.Response.WriteHeader(c.StatusCode)
	if _, err = c.Response.Write([]byte(c.Data["callback"].(string) + "(")); err != nil {
		return
	}
	if _, err = c.Response.Write(b); err != nil {
		return
	}
	_, err = c.Response.Write([]byte(");"))
	return
}

// XML sends an XML response with `Context#StatusCode` and `Context#Data["xml"]`.
func (c *Context) XML() (err error) {
	data := c.Data["xml"]
	b, err := xml.Marshal(data)
	if c.Air.Debug {
		b, err = xml.MarshalIndent(data, "", "\t")
	}
	if err != nil {
		return err
	}
	return c.XMLBlob(b)
}

// XMLBlob sends a XML blob response with `Context#StatusCode`.
func (c *Context) XMLBlob(b []byte) (err error) {
	c.Response.Header.Set(HeaderContentType, MIMEApplicationXML)
	c.Response.WriteHeader(c.StatusCode)
	if _, err = c.Response.Write([]byte(xml.Header)); err != nil {
		return
	}
	_, err = c.Response.Write(b)
	return
}

// File sends a response with the content of the file.
func (c *Context) File(file string) error {
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
		if fi, err = f.Stat(); err != nil {
			return err
		}
	}
	return c.ServeContent(f, fi.Name(), fi.ModTime())
}

// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
// client to save the file.
func (c *Context) Attachment(r io.ReadSeeker, name string) (err error) {
	c.Response.Header.Set(HeaderContentType, ContentTypeByExtension(name))
	c.Response.Header.Set(HeaderContentDisposition, "attachment; filename="+name)
	c.Response.WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response, r)
	return
}

// NoContent sends a response with no body and a `Context#StatusCode`.
func (c *Context) NoContent() error {
	c.Response.WriteHeader(c.StatusCode)
	return nil
}

// Redirect redirects the request with `Context#StatusCode`.
func (c *Context) Redirect(uri string) error {
	if c.StatusCode < http.StatusMultipleChoices || c.StatusCode > http.StatusTemporaryRedirect {
		return ErrInvalidRedirectCode
	}
	c.Response.Header.Set(HeaderLocation, uri)
	c.Response.WriteHeader(c.StatusCode)
	return nil
}

// Error invokes the registered HTTP error handler. Generally used by gas.
func (c *Context) Error(err error) {
	c.Air.HTTPErrorHandler(err, c)
}

// ServeContent sends static content from `io.Reader` and handles caching
// via `If-Modified-Since` request header. It automatically sets `Content-Type`
// and `Last-Modified` response headers.
func (c *Context) ServeContent(content io.ReadSeeker, name string, modtime time.Time) error {
	req := c.Request
	res := c.Response

	if t, err := time.Parse(http.TimeFormat, req.Header.Get(HeaderIfModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		res.Header.Del(HeaderContentType)
		res.Header.Del(HeaderContentLength)
		c.StatusCode = http.StatusNotModified
		return c.NoContent()
	}

	res.Header.Set(HeaderContentType, ContentTypeByExtension(name))
	res.Header.Set(HeaderLastModified, modtime.UTC().Format(http.TimeFormat))
	res.WriteHeader(http.StatusOK)
	_, err := io.Copy(res, content)
	return err
}

// Reset resets the context after request completes. It must be called along
// with `Air#AcquireContext()` and `Air#ReleaseContext()`.
// See `Air#ServeHTTP()`
func (c *Context) Reset(req *Request, res *Response) {
	c.goContext = context.Background()
	c.Request = req
	c.Response = res
	c.Handler = NotFoundHandler
}

// ContentTypeByExtension returns the MIME type associated with the file based on
// its extension. It returns `application/octet-stream` incase MIME type is not
// found.
func ContentTypeByExtension(name string) (t string) {
	if t = mime.TypeByExtension(filepath.Ext(name)); t == "" {
		t = MIMEOctetStream
	}
	return
}

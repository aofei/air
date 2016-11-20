package air

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"
)

// Context represents the context of the current HTTP request.
// It's embedded with `context.Context`.
type Context struct {
	context.Context

	Request  *Request
	Response *Response

	PristinePath string
	ParamNames   []string
	ParamValues  []string
	Params       map[string]string
	Handler      HandlerFunc

	Air *Air

	// MARK: Alias fields for `Response`.

	// Data is an alias for `Response#Data`.
	Data JSONMap
}

var contextPool *sync.Pool

// newContext returns a pointer of a new instance of `Context`.
func newContext(a *Air) *Context {
	c := &Context{}
	c.Context = context.Background()
	c.Request = newRequest(c)
	c.Response = newResponse(c)
	c.Params = make(map[string]string)
	c.Handler = NotFoundHandler
	c.Air = a
	c.Data = c.Response.Data
	return c
}

// SetValue sets request-scoped value into the `Context` of the c.
func (c *Context) SetValue(key interface{}, val interface{}) {
	c.Context = context.WithValue(c.Context, key, val)
}

// feed feeds the rw and req into where they should be.
func (c *Context) feed(rw http.ResponseWriter, req *http.Request) {
	c.Request.Request = req
	c.Request.URL.URL = req.URL
	c.Response.ResponseWriter = rw
}

// reset resets all fields in the c.
func (c *Context) reset() {
	c.Context = context.Background()
	c.Request.reset()
	c.Response.reset()
	c.PristinePath = ""
	c.ParamNames = c.ParamNames[:0]
	c.ParamValues = c.ParamValues[:0]
	for k := range c.Params {
		delete(c.Params, k)
	}
	c.Handler = NotFoundHandler
	c.Data = c.Response.Data
}

// MARK: Alias methods for the `Context#Request`.

// Bind is an alias for the `Request#Bind()` of the c.
func (c *Context) Bind(i interface{}) error {
	return c.Request.Bind(i)
}

// QueryValue is an alias for the `Request#QueryValue()` of the c.
func (c *Context) QueryValue(key string) string {
	return c.Request.QueryValue(key)
}

// QueryValues is an alias for the `Request#QueryValues()` of the c.
func (c *Context) QueryValues() url.Values {
	return c.Request.QueryValues()
}

// FormValue is an alias for the `Request#FormValue()` of the c.
func (c *Context) FormValue(key string) string {
	return c.Request.FormValue(key)
}

// FormValues is an alias for the `Request#FormValues()` of the c.
func (c *Context) FormValues() (url.Values, error) {
	return c.Request.FormValues()
}

// FormFile is an alias for the `Request#FormFile()` of the c.
func (c *Context) FormFile(key string) (multipart.File, *multipart.FileHeader,
	error) {
	return c.Request.FormFile(key)
}

// Cookie is an alias for the `Request#Cookie()` of the c.
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// Cookies is an alias for the `Request#Cookies()` of the c.
func (c *Context) Cookies() []*http.Cookie {
	return c.Request.Cookies()
}

// MARK: Alias methods for the `Context#Response`.

// SetCookie is an alias for the `Response#SetCookie()` of the c.
func (c *Context) SetCookie(cookie *http.Cookie) {
	c.Response.SetCookie(cookie)
}

// Render is an alias for the `Response#Render()` of the c.
func (c *Context) Render() error {
	return c.Response.Render()
}

// HTML is an alias for the `Response#HTML()` of the c.
func (c *Context) HTML() error {
	return c.Response.HTML()
}

// String is an alias for the `Response#String()` of the c.
func (c *Context) String() error {
	return c.Response.String()
}

// JSON is an alias for the `Response#JSON()` of the c.
func (c *Context) JSON() error {
	return c.Response.JSON()
}

// JSONBlob is an alias for the `Response#JSONBlob()` of the c.
func (c *Context) JSONBlob(b []byte) error {
	return c.Response.JSONBlob(b)
}

// JSONP is an alias for the `Response#JSONP()` of the c.
func (c *Context) JSONP() error {
	return c.Response.JSONP()
}

// JSONPBlob is an alias for the `Response#JSONPBlob()` of the c.
func (c *Context) JSONPBlob(b []byte) error {
	return c.Response.JSONPBlob(b)
}

// XML is an alias for the `Response#XML()` of the c.
func (c *Context) XML() error {
	return c.Response.XML()
}

// XMLBlob is an alias for the `Response#XMLBlob()` of the c.
func (c *Context) XMLBlob(b []byte) error {
	return c.Response.XMLBlob(b)
}

// Blob is an alias for the `Response#Blob()` of the c.
func (c *Context) Blob(contentType string, b []byte) error {
	return c.Response.Blob(contentType, b)
}

// Stream is an alias for the `Response#Stream()` of the c.
func (c *Context) Stream(contentType string, r io.Reader) error {
	return c.Response.Stream(contentType, r)
}

// File is an alias for the `Response#File()` of the c.
func (c *Context) File(file string) error {
	return c.Response.File(file)
}

// Attachment is an alias for the `Response#Attachment()` of the c.
func (c *Context) Attachment(file, name string) error {
	return c.Response.Attachment(file, name)
}

// Inline is an alias for the `Response#Inline()` of the c.
func (c *Context) Inline(file, name string) error {
	return c.Response.Inline(file, name)
}

// NoContent is an alias for the `Response#NoContent()` of the c.
func (c *Context) NoContent() error {
	return c.Response.NoContent()
}

// Redirect is an alias for the `Response#Redirect()` of the c.
func (c *Context) Redirect(code int, url string) error {
	return c.Response.Redirect(code, url)
}

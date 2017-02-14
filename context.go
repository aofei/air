package air

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

// Context represents the context of the current HTTP request.
//
// It's embedded with the `context.Context`.
type Context struct {
	context.Context

	Air *Air

	Request  *Request
	Response *Response

	PristinePath string
	ParamNames   []string
	ParamValues  []string
	Handler      Handler

	// Cancel is non-nil if one of the `SetCancel()`, the `SetDeadline()` or the `SetTimeout()`
	// is called. It will be called when the HTTP server finishes the current cycle if it is
	// non-nil and has never been called.
	Cancel context.CancelFunc

	// MARK: Alias fields for the `Response`.

	// Data is an alias for the `Response#Data`.
	Data JSONMap
}

// NewContext returns a pointer of a new instance of the `Context`.
func NewContext(a *Air) *Context {
	c := &Context{Air: a}
	c.Request = NewRequest(c)
	c.Response = NewResponse(c)
	c.ParamValues = make([]string, 0, a.paramCap)
	c.Handler = NotFoundHandler
	c.Data = c.Response.Data
	return c
}

// SetCancel sets a new done channel into the `Context` of the c.
func (c *Context) SetCancel() {
	c.Context, c.Cancel = context.WithCancel(c.Context)
}

// SetDeadline sets a new deadline into the `Context` of the c.
func (c *Context) SetDeadline(deadline time.Time) {
	c.Context, c.Cancel = context.WithDeadline(c.Context, deadline)
}

// SetTimeout sets a new deadline based on the timeout into the `Context` of the c.
func (c *Context) SetTimeout(timeout time.Duration) {
	c.Context, c.Cancel = context.WithTimeout(c.Context, timeout)
}

// SetValue sets request-scoped value into the `Context` of the c.
func (c *Context) SetValue(key interface{}, val interface{}) {
	c.Context = context.WithValue(c.Context, key, val)
}

// Param returns the path param value by the name.
func (c *Context) Param(name string) string {
	for i, n := range c.ParamNames {
		if n == name {
			return c.ParamValues[i]
		}
	}
	return ""
}

// feed feeds the req and the rw into where they should be.
func (c *Context) feed(req *http.Request, rw http.ResponseWriter) {
	c.Context = req.Context()
	c.Request.feed(req)
	c.Response.feed(rw)
}

// reset resets all fields in the c.
func (c *Context) reset() {
	c.Context = nil
	c.Request.reset()
	c.Response.reset()
	c.PristinePath = ""
	c.ParamNames = c.ParamNames[:0]
	c.ParamValues = c.ParamValues[:0]
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
func (c *Context) FormValues() url.Values {
	return c.Request.FormValues()
}

// FormFile is an alias for the `Request#FormFile()` of the c.
func (c *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
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

// Push is an alias for the `Response#Push()` of the c.
func (c *Context) Push(target string, pos *http.PushOptions) error {
	return c.Response.Push(target, pos)
}

// Render is an alias for the `Response#Render()` of the c.
func (c *Context) Render(templates ...string) error {
	return c.Response.Render(templates...)
}

// HTML is an alias for the `Response#HTML()` of the c.
func (c *Context) HTML(html string) error {
	return c.Response.HTML(html)
}

// String is an alias for the `Response#String()` of the c.
func (c *Context) String(s string) error {
	return c.Response.String(s)
}

// JSON is an alias for the `Response#JSON()` of the c.
func (c *Context) JSON(i interface{}) error {
	return c.Response.JSON(i)
}

// JSONP is an alias for the `Response#JSONP()` of the c.
func (c *Context) JSONP(i interface{}, callback string) error {
	return c.Response.JSONP(i, callback)
}

// XML is an alias for the `Response#XML()` of the c.
func (c *Context) XML(i interface{}) error {
	return c.Response.XML(i)
}

// YAML is an alias for the `Response#YAML()` of the c.
func (c *Context) YAML(i interface{}) error {
	return c.Response.YAML(i)
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
func (c *Context) Attachment(file, filename string) error {
	return c.Response.Attachment(file, filename)
}

// Inline is an alias for the `Response#Inline()` of the c.
func (c *Context) Inline(file, filename string) error {
	return c.Response.Inline(file, filename)
}

// NoContent is an alias for the `Response#NoContent()` of the c.
func (c *Context) NoContent() error {
	return c.Response.NoContent()
}

// Redirect is an alias for the `Response#Redirect()` of the c.
func (c *Context) Redirect(statusCode int, url string) error {
	return c.Response.Redirect(statusCode, url)
}

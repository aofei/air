package air

import (
	"bufio"
	"context"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Context represents the context of the current HTTP request.
//
// It's embedded with the `context.Context`.
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

	// MARK: Alias fields for the `Response`.

	// Data is an alias for the `Response#Data`.
	Data JSONMap
}

var contextPool *sync.Pool

// newContext returns a pointer of a new instance of the `Context`.
func newContext(a *Air) *Context {
	c := &Context{}
	c.Request = newRequest(c)
	c.Response = newResponse(c)
	c.Params = make(map[string]string)
	c.Handler = NotFoundHandler
	c.Air = a
	c.Data = c.Response.Data
	return c
}

// SetCancel sets a new done channel into the `Context` of the c.
func (c *Context) SetCancel() {
	c.Context, _ = context.WithCancel(c.Context)
}

// SetDeadline sets a new deadline into the `Context` of the c.
func (c *Context) SetDeadline(deadline time.Time) {
	c.Context, _ = context.WithDeadline(c.Context, deadline)
}

// SetTimeout sets a new deadline based on the timeout into the `Context` of the c.
func (c *Context) SetTimeout(timeout time.Duration) {
	c.Context, _ = context.WithTimeout(c.Context, timeout)
}

// SetValue sets request-scoped value into the `Context` of the c.
func (c *Context) SetValue(key interface{}, val interface{}) {
	c.Context = context.WithValue(c.Context, key, val)
}

// feed feeds the req and the rw into where they should be.
func (c *Context) feed(req *http.Request, rw http.ResponseWriter) {
	c.Context = req.Context()
	c.Request.Request = req
	c.Request.URL.URL = req.URL
	c.Response.ResponseWriter = rw
}

// reset resets all fields in the c.
func (c *Context) reset() {
	c.Context = nil
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

// Hijack is an alias for the `Response#Hijack()` of the c.
func (c *Context) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return c.Response.Hijack()
}

// CloseNotify is an alias for the `Response#CloseNotify()` of the c.
func (c *Context) CloseNotify() <-chan bool {
	return c.Response.CloseNotify()
}

// Flush is an alias for the `Response#Flush()` of the c.
func (c *Context) Flush() {
	c.Response.Flush()
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

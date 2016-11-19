package air

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

// Context represents the context of the current HTTP request. It holds request
// and response writer objects, path, path parameters, data and registered
// handler.
type Context struct {
	context.Context
	http.ResponseWriter

	statusCode int
	size       int

	Request      *http.Request
	PristinePath string
	ParamNames   []string
	ParamValues  []string
	Params       map[string]string
	Handler      HandlerFunc
	Data         JSONMap
	Written      bool
	Air          *Air
}

var contextPool *sync.Pool

// newContext returns a new instance of `Context`.
func newContext(a *Air) *Context {
	return &Context{
		Context: context.Background(),
		Params:  make(map[string]string),
		Handler: NotFoundHandler,
		Data:    make(JSONMap),
		Air:     a,
	}
}

// SetValue sets request-scoped value into the `Context` of the c.
func (c *Context) SetValue(key interface{}, val interface{}) {
	c.Context = context.WithValue(c.Context, key, val)
}

// Write implements `http.ResponseWriter#Write()`.
func (c *Context) Write(bs []byte) (int, error) {
	if !c.Written {
		c.WriteHeader(http.StatusOK)
	}
	n, err := c.ResponseWriter.Write(bs)
	c.size += n
	return n, err
}

// WriteHeader implements `http.ResponseWriter#WriteHeader()`.
func (c *Context) WriteHeader(code int) {
	if c.Written {
		c.Air.Logger.Warn("response already committed")
		return
	}
	c.statusCode = code
	c.ResponseWriter.WriteHeader(code)
	c.Written = true
}

// StatusCode returns the HTTP status code.
func (c *Context) StatusCode() int {
	return c.statusCode
}

// Size returns the number of bytes already written into the response HTTP body.
func (c *Context) Size() int {
	return c.size
}

// SetCookie adds a "Set-Cookie" header in HTTP response. The provided cookie
// must have a valid `Name`. Invalid cookies may be silently dropped.
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.ResponseWriter, cookie)
}

// Bind binds the request body into provided type i. The default binder does it
// based on "Content-Type" header.
func (c *Context) Bind(i interface{}) error {
	return c.Air.binder.bind(i, c)
}

// Render renders a template with `Data` and `Data["template"]` of the c and
// sends a "text/html" response with `StatusCode` of the c.
func (c *Context) Render() error {
	t, ok := c.Data["template"]
	if !ok || reflect.ValueOf(t).Kind() != reflect.String {
		return errors.New("c.Data[\"template\"] not setted")
	}
	buf := &bytes.Buffer{}
	if err := c.Air.renderer.render(buf, t.(string), c); err != nil {
		return err
	}
	c.Header().Set(HeaderContentType, MIMETextHTML)
	_, err := c.Write(buf.Bytes())
	return err
}

// HTML sends an HTTP response with `StatusCode` and `Data["html"]` of the c.
func (c *Context) HTML() error {
	h, ok := c.Data["html"]
	if !ok || reflect.ValueOf(h).Kind() != reflect.String {
		return errors.New("c.Data[\"html\"] not setted")
	}
	c.Header().Set(HeaderContentType, MIMETextHTML)
	_, err := c.Write([]byte(h.(string)))
	return err
}

// String sends a string response with `StatusCode` and `Data["string"]` of the
// c.
func (c *Context) String() error {
	s, ok := c.Data["string"]
	if !ok || reflect.ValueOf(s).Kind() != reflect.String {
		return errors.New("c.Data[\"string\"] not setted")
	}
	c.Header().Set(HeaderContentType, MIMETextPlain)
	_, err := c.Write([]byte(s.(string)))
	return err
}

// JSON sends a JSON response with `StatusCode` and `Data["json"]` of the c.
func (c *Context) JSON() error {
	j, ok := c.Data["json"]
	if !ok {
		return errors.New("c.Data[\"json\"] not setted")
	}
	bs, err := json.Marshal(j)
	if c.Air.Config.DebugMode {
		bs, err = json.MarshalIndent(j, "", "\t")
	}
	if err != nil {
		return err
	}
	return c.JSONBlob(bs)
}

// JSONBlob sends a JSON blob response with `StatusCode` of the c.
func (c *Context) JSONBlob(bs []byte) error {
	return c.Blob(MIMEApplicationJSON, bs)
}

// JSONP sends a JSONP response with `StatusCode` and `Data["jsonp"]` of the c.
// It uses `Data["callback"]` of the c to construct the JSONP payload.
func (c *Context) JSONP() error {
	j, jok := c.Data["jsonp"]
	if !jok {
		return errors.New("c.Data[\"jsonp\"] not setted")
	}
	bs, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return c.JSONPBlob(bs)
}

// JSONPBlob sends a JSONP blob response with `StatusCode` of the c. It uses
// `Data["callback"]` of the c to construct the JSONP payload.
func (c *Context) JSONPBlob(bs []byte) error {
	cb, cbok := c.Data["callback"]
	if !cbok || reflect.ValueOf(cb).Kind() != reflect.String {
		return errors.New("c.Data[\"callback\"] not setted")
	}
	c.Header().Set(HeaderContentType, MIMEApplicationJavaScript)
	if _, err := c.Write([]byte(cb.(string) + "(")); err != nil {
		return err
	}
	if _, err := c.Write(bs); err != nil {
		return err
	}
	_, err := c.Write([]byte(");"))
	return err
}

// XML sends an XML response with `StatusCode` and `Data["xml"]` of the c.
func (c *Context) XML() error {
	x, ok := c.Data["xml"]
	if !ok {
		return errors.New("c.Data[\"xml\"] not setted")
	}
	bs, err := xml.Marshal(x)
	if c.Air.Config.DebugMode {
		bs, err = xml.MarshalIndent(x, "", "\t")
	}
	if err != nil {
		return err
	}
	return c.XMLBlob(bs)
}

// XMLBlob sends a XML blob response with `StatusCode` of the c.
func (c *Context) XMLBlob(bs []byte) error {
	if _, err := c.Write([]byte(xml.Header)); err != nil {
		return err
	}
	return c.Blob(MIMEApplicationXML, bs)
}

// Blob sends a blob response with `StatusCode` of the c and contentType.
func (c *Context) Blob(contentType string, bs []byte) error {
	c.Header().Set(HeaderContentType, contentType)
	_, err := c.Write(bs)
	return err
}

// Stream sends a streaming response with `StatusCode` of the c and contentType.
func (c *Context) Stream(contentType string, r io.Reader) error {
	c.Header().Set(HeaderContentType, contentType)
	_, err := io.Copy(c, r)
	return err
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
		defer f.Close()
		if fi, err = f.Stat(); err != nil {
			return err
		}
	}
	http.ServeContent(c, c.Request, fi.Name(), fi.ModTime(), f)
	return nil
}

// Attachment sends a response as attachment, prompting client to save the file.
func (c *Context) Attachment(file, name string) error {
	return c.contentDisposition(file, name, "attachment")
}

// Inline sends a response as inline, opening the file in the browser.
func (c *Context) Inline(file, name string) error {
	return c.contentDisposition(file, name, "inline")
}

// contentDisposition sends a response as the dispositionType.
func (c *Context) contentDisposition(file, name, dispositionType string) error {
	c.Header().Set(HeaderContentDisposition,
		fmt.Sprintf("%s; filename=%s", dispositionType, name))
	return c.File(file)
}

// NoContent sends a response with no body.
func (c *Context) NoContent() error { return nil }

// Redirect redirects the request with status code.
func (c *Context) Redirect(code int, url string) error {
	if code < http.StatusMultipleChoices ||
		code > http.StatusTemporaryRedirect {
		return ErrInvalidRedirectCode
	}
	c.Header().Set(HeaderLocation, url)
	c.WriteHeader(code)
	return nil
}

// reset resets all fields in the c.
func (c *Context) reset() {
	c.Context = context.Background()
	c.ResponseWriter = nil
	c.statusCode = 0
	c.size = 0
	c.Request = nil
	c.PristinePath = ""
	c.ParamNames = c.ParamNames[:0]
	c.ParamValues = c.ParamValues[:0]
	for k := range c.Params {
		delete(c.Params, k)
	}
	c.Handler = NotFoundHandler
	for k := range c.Data {
		delete(c.Data, k)
	}
	c.Written = false
}

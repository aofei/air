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

	"air/log"
)

type (
	// Context represents the context of the current HTTP request. It holds request and
	// response objects, path, path parameters, data and registered handler.
	Context interface {
		// Context returns `net/context.Context`.
		Context() context.Context

		// SetContext sets `net/context.Context`.
		SetContext(context.Context)

		// Deadline returns the time when work done on behalf of this context
		// should be canceled.  Deadline returns ok==false when no deadline is
		// set.  Successive calls to Deadline return the same results.
		Deadline() (deadline time.Time, ok bool)

		// Done returns a channel that's closed when work done on behalf of this
		// context should be canceled.  Done may return nil if this context can
		// never be canceled.  Successive calls to Done return the same value.
		Done() <-chan struct{}

		// Err returns a non-nil error value after Done is closed.  Err returns
		// Canceled if the context was canceled or DeadlineExceeded if the
		// context's deadline passed.  No other values for Err are defined.
		// After Done is closed, successive calls to Err return the same value.
		Err() error

		// Value returns the value associated with this context for key, or nil
		// if no value is associated with key.  Successive calls to Value with
		// the same key returns the same result.
		Value(key interface{}) interface{}

		// Request returns `Request` interface.
		Request() Request

		// Request returns `Response` interface.
		Response() Response

		// Path returns the registered path for the handler.
		Path() string

		// SetPath sets the registered path for the handler.
		SetPath(string)

		// P returns path parameter by index.
		P(int) string

		// Param returns path parameter by name.
		Param(string) string

		// ParamNames returns path parameter names.
		ParamNames() []string

		// SetParamNames sets path parameter names.
		SetParamNames(...string)

		// ParamValues returns path parameter values.
		ParamValues() []string

		// SetParamValues sets path parameter values.
		SetParamValues(...string)

		// QueryParam returns the query param for the provided name. It is an alias
		// for `URL#QueryParam()`.
		QueryParam(string) string

		// QueryParams returns the query parameters as map.
		// It is an alias for `URL#QueryParams()`.
		QueryParams() map[string][]string

		// FormValue returns the form field value for the provided name. It is an
		// alias for `Request#FormValue()`.
		FormValue(string) string

		// FormParams returns the form parameters as map.
		// It is an alias for `Request#FormParams()`.
		FormParams() map[string][]string

		// FormFile returns the multipart form file for the provided name. It is an
		// alias for `Request#FormFile()`.
		FormFile(string) (*multipart.FileHeader, error)

		// MultipartForm returns the multipart form.
		// It is an alias for `Request#MultipartForm()`.
		MultipartForm() (*multipart.Form, error)

		// Cookie returns the named cookie provided in the request.
		// It is an alias for `Request#Cookie()`.
		Cookie(string) (Cookie, error)

		// SetCookie adds a `Set-Cookie` header in HTTP response.
		// It is an alias for `Response#SetCookie()`.
		SetCookie(Cookie)

		// Cookies returns the HTTP cookies sent with the request.
		// It is an alias for `Request#Cookies()`.
		Cookies() []Cookie

		// Get retrieves data from the context.
		Get(string) interface{}

		// Set saves data in the context.
		Set(string, interface{})

		// Bind binds the request body into provided type `i`. The default binder
		// does it based on Content-Type header.
		Bind(interface{}) error

		// Render renders a template with data and sends a text/html response with status
		// code. Templates can be registered using `Air.SetRenderer()`.
		Render(int, string, interface{}) error

		// HTML sends an HTTP response with status code.
		HTML(int, string) error

		// String sends a string response with status code.
		String(int, string) error

		// JSON sends a JSON response with status code.
		JSON(int, interface{}) error

		// JSONBlob sends a JSON blob response with status code.
		JSONBlob(int, []byte) error

		// JSONP sends a JSONP response with status code. It uses `callback` to construct
		// the JSONP payload.
		JSONP(int, string, interface{}) error

		// XML sends an XML response with status code.
		XML(int, interface{}) error

		// XMLBlob sends a XML blob response with status code.
		XMLBlob(int, []byte) error

		// File sends a response with the content of the file.
		File(string) error

		// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
		// client to save the file.
		Attachment(io.ReadSeeker, string) error

		// NoContent sends a response with no body and a status code.
		NoContent(int) error

		// Redirect redirects the request with status code.
		Redirect(int, string) error

		// Error invokes the registered HTTP error handler. Generally used by gas.
		Error(err error)

		// Handler returns the matched handler by router.
		Handler() HandlerFunc

		// SetHandler sets the matched handler by router.
		SetHandler(HandlerFunc)

		// Logger returns the `Logger` instance.
		Logger() *log.Logger

		// Air returns the `Air` instance.
		Air() *Air

		// ServeContent sends static content from `io.Reader` and handles caching
		// via `If-Modified-Since` request header. It automatically sets `Content-Type`
		// and `Last-Modified` response headers.
		ServeContent(io.ReadSeeker, string, time.Time) error

		// Reset resets the context after request completes. It must be called along
		// with `Air#AcquireContext()` and `Air#ReleaseContext()`.
		// See `Air#ServeHTTP()`
		Reset(Request, Response)
	}

	airContext struct {
		context  context.Context
		request  Request
		response Response
		path     string
		pnames   []string
		pvalues  []string
		handler  HandlerFunc
		air      *Air
	}
)

const (
	indexPage = "index.html"
)

func (c *airContext) Context() context.Context {
	return c.context
}

func (c *airContext) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *airContext) Deadline() (deadline time.Time, ok bool) {
	return c.context.Deadline()
}

func (c *airContext) Done() <-chan struct{} {
	return c.context.Done()
}

func (c *airContext) Err() error {
	return c.context.Err()
}

func (c *airContext) Value(key interface{}) interface{} {
	return c.context.Value(key)
}

func (c *airContext) Request() Request {
	return c.request
}

func (c *airContext) Response() Response {
	return c.response
}

func (c *airContext) Path() string {
	return c.path
}

func (c *airContext) SetPath(p string) {
	c.path = p
}

func (c *airContext) P(i int) (value string) {
	l := len(c.pnames)
	if i < l {
		value = c.pvalues[i]
	}
	return
}

func (c *airContext) Param(name string) (value string) {
	l := len(c.pnames)
	for i, n := range c.pnames {
		if n == name && i < l {
			value = c.pvalues[i]
			break
		}
	}
	return
}

func (c *airContext) ParamNames() []string {
	return c.pnames
}

func (c *airContext) SetParamNames(names ...string) {
	c.pnames = names
}

func (c *airContext) ParamValues() []string {
	return c.pvalues
}

func (c *airContext) SetParamValues(values ...string) {
	c.pvalues = values
}

func (c *airContext) QueryParam(name string) string {
	return c.request.URL().QueryParam(name)
}

func (c *airContext) QueryParams() map[string][]string {
	return c.request.URL().QueryParams()
}

func (c *airContext) FormValue(name string) string {
	return c.request.FormValue(name)
}

func (c *airContext) FormParams() map[string][]string {
	return c.request.FormParams()
}

func (c *airContext) FormFile(name string) (*multipart.FileHeader, error) {
	return c.request.FormFile(name)
}

func (c *airContext) MultipartForm() (*multipart.Form, error) {
	return c.request.MultipartForm()
}

func (c *airContext) Cookie(name string) (Cookie, error) {
	return c.request.Cookie(name)
}

func (c *airContext) SetCookie(cookie Cookie) {
	c.response.SetCookie(cookie)
}

func (c *airContext) Cookies() []Cookie {
	return c.request.Cookies()
}

func (c *airContext) Set(key string, val interface{}) {
	c.context = context.WithValue(c.context, key, val)
}

func (c *airContext) Get(key string) interface{} {
	return c.context.Value(key)
}

func (c *airContext) Bind(i interface{}) error {
	return c.air.binder.Bind(i, c)
}

func (c *airContext) Render(code int, name string, data interface{}) (err error) {
	if c.air.renderer == nil {
		return ErrRendererNotRegistered
	}
	buf := new(bytes.Buffer)
	if err = c.air.renderer.Render(buf, name, data, c); err != nil {
		return
	}
	c.response.Header().Set(HeaderContentType, MIMETextHTML)
	c.response.WriteHeader(code)
	_, err = c.response.Write(buf.Bytes())
	return
}

func (c *airContext) HTML(code int, html string) (err error) {
	c.response.Header().Set(HeaderContentType, MIMETextHTML)
	c.response.WriteHeader(code)
	_, err = c.response.Write([]byte(html))
	return
}

func (c *airContext) String(code int, s string) (err error) {
	c.response.Header().Set(HeaderContentType, MIMETextPlain)
	c.response.WriteHeader(code)
	_, err = c.response.Write([]byte(s))
	return
}

func (c *airContext) JSON(code int, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if c.air.Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.JSONBlob(code, b)
}

func (c *airContext) JSONBlob(code int, b []byte) (err error) {
	c.response.Header().Set(HeaderContentType, MIMEApplicationJSON)
	c.response.WriteHeader(code)
	_, err = c.response.Write(b)
	return
}

func (c *airContext) JSONP(code int, callback string, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	c.response.Header().Set(HeaderContentType, MIMEApplicationJavaScript)
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.Write(b); err != nil {
		return
	}
	_, err = c.response.Write([]byte(");"))
	return
}

func (c *airContext) XML(code int, i interface{}) (err error) {
	b, err := xml.Marshal(i)
	if c.air.Debug() {
		b, err = xml.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.XMLBlob(code, b)
}

func (c *airContext) XMLBlob(code int, b []byte) (err error) {
	c.response.Header().Set(HeaderContentType, MIMEApplicationXML)
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(xml.Header)); err != nil {
		return
	}
	_, err = c.response.Write(b)
	return
}

func (c *airContext) File(file string) error {
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

func (c *airContext) Attachment(r io.ReadSeeker, name string) (err error) {
	c.response.Header().Set(HeaderContentType, ContentTypeByExtension(name))
	c.response.Header().Set(HeaderContentDisposition, "attachment; filename="+name)
	c.response.WriteHeader(http.StatusOK)
	_, err = io.Copy(c.response, r)
	return
}

func (c *airContext) NoContent(code int) error {
	c.response.WriteHeader(code)
	return nil
}

func (c *airContext) Redirect(code int, url string) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return ErrInvalidRedirectCode
	}
	c.response.Header().Set(HeaderLocation, url)
	c.response.WriteHeader(code)
	return nil
}

func (c *airContext) Error(err error) {
	c.air.httpErrorHandler(err, c)
}

func (c *airContext) Air() *Air {
	return c.air
}

func (c *airContext) Handler() HandlerFunc {
	return c.handler
}

func (c *airContext) SetHandler(h HandlerFunc) {
	c.handler = h
}

func (c *airContext) Logger() *log.Logger {
	return &c.air.logger
}

func (c *airContext) ServeContent(content io.ReadSeeker, name string, modtime time.Time) error {
	req := c.Request()
	res := c.Response()

	if t, err := time.Parse(http.TimeFormat, req.Header().Get(HeaderIfModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		res.Header().Del(HeaderContentType)
		res.Header().Del(HeaderContentLength)
		return c.NoContent(http.StatusNotModified)
	}

	res.Header().Set(HeaderContentType, ContentTypeByExtension(name))
	res.Header().Set(HeaderLastModified, modtime.UTC().Format(http.TimeFormat))
	res.WriteHeader(http.StatusOK)
	_, err := io.Copy(res, content)
	return err
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

func (c *airContext) Reset(req Request, res Response) {
	c.context = context.Background()
	c.request = req
	c.response = res
	c.handler = NotFoundHandler
}

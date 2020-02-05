package air

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const easterEgg = `
01000010 01100001 01111010 01101001
01101110 01100111 01100001 00100001
`

func TestNew(t *testing.T) {
	a := New()

	assert.Equal(t, "air", a.AppName)
	assert.Empty(t, a.MaintainerEmail)
	assert.False(t, a.DebugMode)
	assert.Equal(t, "localhost:8080", a.Address)
	assert.Zero(t, a.ReadTimeout)
	assert.Zero(t, a.ReadHeaderTimeout)
	assert.Zero(t, a.WriteTimeout)
	assert.Zero(t, a.IdleTimeout)
	assert.Equal(t, 1048576, a.MaxHeaderBytes)
	assert.Empty(t, a.TLSCertFile)
	assert.Empty(t, a.TLSKeyFile)
	assert.False(t, a.ACMEEnabled)
	assert.Equal(
		t,
		"https://acme-v02.api.letsencrypt.org/directory",
		a.ACMEDirectoryURL,
	)
	assert.Equal(t, "acme-certs", a.ACMECertRoot)
	assert.Nil(t, a.ACMEHostWhitelist)
	assert.False(t, a.HTTPSEnforced)
	assert.Equal(t, "80", a.HTTPSEnforcedPort)
	assert.Zero(t, a.WebSocketHandshakeTimeout)
	assert.Nil(t, a.WebSocketSubprotocols)
	assert.False(t, a.PROXYEnabled)
	assert.Zero(t, a.PROXYReadHeaderTimeout)
	assert.Nil(t, a.PROXYRelayerIPWhitelist)
	assert.Nil(t, a.Pregases)
	assert.Nil(t, a.Gases)
	assert.IsType(t, DefaultNotFoundHandler, a.NotFoundHandler)
	assert.IsType(
		t,
		DefaultMethodNotAllowedHandler,
		a.MethodNotAllowedHandler,
	)
	assert.IsType(t, DefaultErrorHandler, a.ErrorHandler)
	assert.Nil(t, a.ErrorLogger)
	assert.False(t, a.AutoPushEnabled)
	assert.False(t, a.MinifierEnabled)
	assert.ElementsMatch(t, a.MinifierMIMETypes, []string{
		"text/html",
		"text/css",
		"application/javascript",
		"application/json",
		"application/xml",
		"image/svg+xml",
	})
	assert.False(t, a.GzipEnabled)
	assert.Equal(t, int64(1024), a.GzipMinContentLength)
	assert.ElementsMatch(t, a.GzipMIMETypes, []string{
		"text/plain",
		"text/html",
		"text/css",
		"application/javascript",
		"application/json",
		"application/xml",
		"application/toml",
		"application/yaml",
		"image/svg+xml",
	})
	assert.Equal(t, gzip.DefaultCompression, a.GzipCompressionLevel)
	assert.Equal(t, 8192, a.GzipFlushThreshold)
	assert.Equal(t, "templates", a.RendererTemplateRoot)
	assert.ElementsMatch(t, a.RendererTemplateExts, []string{".html"})
	assert.Equal(t, "{{", a.RendererTemplateLeftDelim)
	assert.Equal(t, "}}", a.RendererTemplateRightDelim)
	assert.Nil(t, a.RendererTemplateFuncMap)
	assert.False(t, a.CofferEnabled)
	assert.Equal(t, 33554432, a.CofferMaxMemoryBytes)
	assert.Equal(t, "assets", a.CofferAssetRoot)
	assert.ElementsMatch(t, a.CofferAssetExts, []string{
		".html",
		".css",
		".js",
		".json",
		".xml",
		".toml",
		".yaml",
		".yml",
		".svg",
		".jpg",
		".jpeg",
		".png",
		".gif",
	})
	assert.False(t, a.I18nEnabled)
	assert.Equal(t, "locales", a.I18nLocaleRoot)
	assert.Equal(t, "en-US", a.I18nLocaleBase)
	assert.Empty(t, a.ConfigFile)

	assert.NotNil(t, a.server)
	assert.NotNil(t, a.router)
	assert.NotNil(t, a.binder)
	assert.NotNil(t, a.minifier)
	assert.NotNil(t, a.renderer)
	assert.NotNil(t, a.coffer)
	assert.NotNil(t, a.i18n)

	assert.NotNil(t, a.contentTypeSnifferBufferPool)
	assert.IsType(t, []byte{}, a.contentTypeSnifferBufferPool.Get())
	assert.Len(t, a.contentTypeSnifferBufferPool.Get(), 512)

	assert.NotNil(t, a.gzipWriterPool)
	assert.IsType(t, &gzip.Writer{}, a.gzipWriterPool.Get())

	assert.NotNil(t, a.reverseProxyTransport)
	assert.NotNil(t, a.reverseProxyBufferPool)
}

func TestAirGET(t *testing.T) {
	a := New()

	a.GET("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [GET /foobar]")
	})

	req := httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [GET /foobar]", rec.Body.String())
}

func TestAirHEAD(t *testing.T) {
	a := New()

	a.HEAD("/foobar", func(req *Request, res *Response) error {
		return res.Write(nil)
	})

	req := httptest.NewRequest(http.MethodHead, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestAirPOST(t *testing.T) {
	a := New()

	a.POST("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [POST /foobar]")
	})

	req := httptest.NewRequest(http.MethodPost, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [POST /foobar]", rec.Body.String())
}

func TestAirPUT(t *testing.T) {
	a := New()

	a.PUT("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [PUT /foobar]")
	})

	req := httptest.NewRequest(http.MethodPut, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [PUT /foobar]", rec.Body.String())
}

func TestAirPATCH(t *testing.T) {
	a := New()

	a.PATCH("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [PATCH /foobar]")
	})

	req := httptest.NewRequest(http.MethodPatch, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [PATCH /foobar]", rec.Body.String())
}

func TestAirDELETE(t *testing.T) {
	a := New()

	a.DELETE("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [DELETE /foobar]")
	})

	req := httptest.NewRequest(http.MethodDelete, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [DELETE /foobar]", rec.Body.String())
}

func TestAirCONNECT(t *testing.T) {
	a := New()

	a.CONNECT("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [CONNECT /foobar]")
	})

	req := httptest.NewRequest(http.MethodConnect, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [CONNECT /foobar]", rec.Body.String())
}

func TestAirOPTIONS(t *testing.T) {
	a := New()

	a.OPTIONS("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [OPTIONS /foobar]")
	})

	req := httptest.NewRequest(http.MethodOptions, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [OPTIONS /foobar]", rec.Body.String())
}

func TestAirTRACE(t *testing.T) {
	a := New()

	a.TRACE("/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [TRACE /foobar]")
	})

	req := httptest.NewRequest(http.MethodTrace, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [TRACE /foobar]", rec.Body.String())
}

func TestAirBATCH(t *testing.T) {
	a := New()

	a.BATCH(nil, "/foobar", func(req *Request, res *Response) error {
		return res.WriteString("Matched [* /foobar]")
	})

	req := httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodPost, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPut, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPatch, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodDelete, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodConnect, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodOptions, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodTrace, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())
}

func TestAirFILE(t *testing.T) {
	a := New()

	f, err := ioutil.TempFile("", "air.TestAirFILE")
	assert.NoError(t, err)
	assert.NotNil(t, f)
	defer os.Remove(f.Name())

	_, err = f.Write([]byte("Foobar"))
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	a.FILE("/foobar", f.Name())

	req := httptest.NewRequest(http.MethodGet, "/foobar", nil)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	a.FILE("/foobar2", "nowhere")

	req = httptest.NewRequest(http.MethodGet, "/foobar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, http.StatusText(rec.Code), rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar2", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestAirFILES(t *testing.T) {
	a := New()

	dir, err := ioutil.TempDir("", "air.TestAirFILES")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	f, err := ioutil.TempFile(dir, "")
	assert.NoError(t, err)
	assert.NotNil(t, f)

	_, err = f.Write([]byte("Foobar"))
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	f2, err := ioutil.TempFile(dir, "")
	assert.NoError(t, err)
	assert.NotNil(t, f2)

	_, err = f2.Write([]byte("Foobar2"))
	assert.NoError(t, err)
	assert.NoError(t, f2.Close())

	a.FILES("/foobar", dir)

	req := httptest.NewRequest(
		http.MethodGet,
		path.Join("/foobar", filepath.Base(f.Name())),
		nil,
	)
	rec := httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar", rec.Body.String())

	req = httptest.NewRequest(
		http.MethodHead,
		path.Join("/foobar", filepath.Base(f.Name())),
		nil,
	)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/foobar/nowhere", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, http.StatusText(rec.Code), rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar/nowhere", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.String())

	a.FILES("/foobar2/", "")

	req = httptest.NewRequest(http.MethodGet, "/foobar2/air.go", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar2/air.go", nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestAirGroup(t *testing.T) {
	a := New()

	g := a.Group("/foobar")
	assert.NotNil(t, g)
	assert.Equal(t, a, g.Air)
	assert.Equal(t, "/foobar", g.Prefix)
	assert.Nil(t, g.Gases)
}

func TestAirServe(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Close())

	dir, err := ioutil.TempDir("", "air.TestAirServe")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a = New()
	a.Address = "localhost:0"
	a.ConfigFile = filepath.Join(dir, "config.json")

	assert.NoError(t, ioutil.WriteFile(
		a.ConfigFile,
		[]byte(`{"app_name":"foobar"}`),
		os.ModePerm,
	))

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.Equal(t, "foobar", a.AppName)

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "localhost:0"
	a.ConfigFile = filepath.Join(dir, "config.toml")

	assert.NoError(t, ioutil.WriteFile(
		a.ConfigFile,
		[]byte(`app_name = "foobar"`),
		os.ModePerm,
	))

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.Equal(t, "foobar", a.AppName)

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "localhost:0"
	a.ConfigFile = filepath.Join(dir, "config.yaml")

	assert.NoError(t, ioutil.WriteFile(
		a.ConfigFile,
		[]byte(`app_name: "foobar"`),
		os.ModePerm,
	))

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.Equal(t, "foobar", a.AppName)

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "localhost:0"
	a.ConfigFile = filepath.Join(dir, "config.yml")

	assert.NoError(t, ioutil.WriteFile(
		a.ConfigFile,
		[]byte(`app_name: "foobar"`),
		os.ModePerm,
	))

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.Equal(t, "foobar", a.AppName)

	assert.NoError(t, a.Close())

	a = New()
	a.ConfigFile = "nowhere"
	assert.True(t, os.IsNotExist(a.Serve()))

	a = New()
	a.ConfigFile = filepath.Join(dir, "config.ext")

	assert.NoError(t, ioutil.WriteFile(a.ConfigFile, nil, os.ModePerm))
	assert.Equal(
		t,
		"air: unsupported configuration file extension: .ext",
		a.Serve().Error(),
	)

	a = New()
	a.ConfigFile = filepath.Join(dir, "config.json")

	assert.NoError(t, ioutil.WriteFile(
		a.ConfigFile,
		[]byte(`{"app_name":0}`),
		os.ModePerm,
	))
	assert.Error(t, a.Serve())
}

func TestAirClose(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Close())
}

func TestAirShutdown(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	hijackOSStdout()

	foo := ""
	a.AddShutdownJob(func() {
		foo = "bar"
	})

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Shutdown(context.Background()))

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, "bar", foo)
}

func TestAirAddShutdownJob(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	hijackOSStdout()

	foo := ""
	id := a.AddShutdownJob(func() {
		foo = "bar"
	})

	assert.Equal(t, 0, id)

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Shutdown(context.Background()))

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, "bar", foo)
}

func TestAirRemoveShutdownJob(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	hijackOSStdout()

	foo := ""
	id := a.AddShutdownJob(func() {
		foo = "bar"
	})

	assert.Equal(t, 0, id)

	a.RemoveShutdownJob(id)

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Shutdown(context.Background()))

	time.Sleep(100 * time.Millisecond)

	assert.Empty(t, foo)
}

func TestAirAddresses(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.Len(t, a.Addresses(), 1)

	assert.NoError(t, a.Close())
}

func TestAirLogErrorf(t *testing.T) {
	a := New()

	buf := bytes.Buffer{}

	log.SetOutput(&buf)
	log.SetFlags(0)
	a.logErrorf("air: some error: %v", errors.New("foobar"))
	assert.Equal(t, buf.String(), "air: some error: foobar\n")
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	buf.Reset()

	a.ErrorLogger = log.New(&buf, "", 0)
	a.logErrorf("air: some error: %v", errors.New("foobar"))
	assert.Equal(t, buf.String(), "air: some error: foobar\n")
}

func TestWrapHTTPHandler(t *testing.T) {
	a := New()

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.NoError(t, WrapHTTPHandler(http.HandlerFunc(func(
		rw http.ResponseWriter,
		r *http.Request,
	) {
		rw.Write([]byte("Foobar"))
	}))(req, res))
	assert.Equal(t, "Foobar", rec.Body.String())
}

func TestDefaultNotFoundHandler(t *testing.T) {
	a := New()

	req, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	err := DefaultNotFoundHandler(req, res)
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusNotFound, res.Status)
	assert.Equal(t, http.StatusText(res.Status), err.Error())
}

func TestDefaultMethodNotAllowedHandler(t *testing.T) {
	a := New()

	req, res, _ := fakeRRCycle(a, http.MethodGet, "/", nil)
	err := DefaultMethodNotAllowedHandler(req, res)
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, res.Status)
	assert.Equal(t, http.StatusText(res.Status), err.Error())
}

func TestDefaultErrorHandler(t *testing.T) {
	a := New()

	req, res, rec := fakeRRCycle(a, http.MethodGet, "/", nil)
	res.Status = http.StatusBadRequest
	DefaultErrorHandler(errors.New("foobar"), req, res)
	assert.Equal(t, "foobar", rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/", nil)
	res.Status = http.StatusInternalServerError
	DefaultErrorHandler(errors.New("foobar"), req, res)
	assert.Equal(t, http.StatusText(res.Status), rec.Body.String())

	req, res, rec = fakeRRCycle(a, http.MethodGet, "/", nil)
	assert.NoError(t, res.WriteString("everything is fine"))
	DefaultErrorHandler(errors.New("foobar"), req, res)
	assert.Equal(t, "everything is fine", rec.Body.String())
}

func TestWrapHTTPMiddleWare(t *testing.T) {
	a := New()

	req, res, rec := fakeRRCycle(a, http.MethodHead, "/", nil)
	assert.NoError(t, WrapHTTPMiddleware(func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			rw http.ResponseWriter,
			r *http.Request,
		) {
			r.Method = http.MethodGet
			next.ServeHTTP(rw, r)
		})
	})(func(req *Request, res *Response) error {
		return res.WriteString("Foobar")
	})(req, res))
	assert.Equal(t, http.MethodGet, req.Method)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar", rec.Body.String())
}

func TestStringSliceContains(t *testing.T) {
	assert.True(t, stringSliceContains([]string{"foo"}, "foo"))
	assert.False(t, stringSliceContains([]string{"foo"}, "bar"))
}

func TestStringSliceContainsCIly(t *testing.T) {
	assert.True(t, stringSliceContainsCIly([]string{"foo"}, "FOO"))
	assert.False(t, stringSliceContainsCIly([]string{"foo"}, "BAR"))
}

func TestSplitPathQuery(t *testing.T) {
	p, q := splitPathQuery("/foobar")
	assert.Equal(t, "/foobar", p)
	assert.Empty(t, q)

	p, q = splitPathQuery("/foobar?")
	assert.Equal(t, "/foobar", p)
	assert.Empty(t, q)

	p, q = splitPathQuery("/foobar?foo=bar")
	assert.Equal(t, "/foobar", p)
	assert.Equal(t, "foo=bar", q)
}

func fakeRRCycle(
	a *Air,
	method string,
	target string,
	body io.Reader,
) (*Request, *Response, *httptest.ResponseRecorder) {
	req := &Request{
		Air: a,

		parseRouteParamsOnce: &sync.Once{},
		parseOtherParamsOnce: &sync.Once{},
	}
	req.SetHTTPRequest(httptest.NewRequest(method, target, body))

	rec := httptest.NewRecorder()
	res := &Response{
		Air:    a,
		Status: http.StatusOK,

		req:  req,
		ohrw: rec,
	}
	res.SetHTTPResponseWriter(&responseWriter{
		r:  res,
		rw: rec,
	})

	req.res = res

	return req, res, rec
}

var (
	osStdout     = os.Stdout
	fakeOSStdout *os.File
)

func hijackOSStdout() {
	if fakeOSStdout == nil {
		fakeOSStdout, _ = ioutil.TempFile("", "")
	}

	os.Stdout = fakeOSStdout
}

func revertOSStdout() {
	os.Stdout = osStdout
}

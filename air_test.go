package air

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	assert.Nil(t, a.ACMEAccountKey)
	assert.Equal(
		t,
		"https://acme-v02.api.letsencrypt.org/directory",
		a.ACMEDirectoryURL,
	)
	assert.Nil(t, a.ACMETOSURLWhitelist)
	assert.Equal(t, "acme-certs", a.ACMECertRoot)
	assert.Empty(t, a.ACMEDefaultHost)
	assert.Nil(t, a.ACMEHostWhitelist)
	assert.Equal(t, 30*24*time.Hour, a.ACMERenewalWindow)
	assert.Nil(t, a.ACMEExtraExts)
	assert.False(t, a.HTTPSEnforced)
	assert.Equal(t, "0", a.HTTPSEnforcedPort)
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
	assert.NotNil(t, a.renderer)
	assert.NotNil(t, a.minifier)
	assert.NotNil(t, a.coffer)
	assert.NotNil(t, a.i18n)

	assert.NotNil(t, a.addressMap)
	assert.Nil(t, a.shutdownJobs)
	assert.NotNil(t, a.shutdownJobMutex)
	assert.Zero(t, cap(a.shutdownJobDone))
	assert.NotNil(t, a.requestPool)
	assert.NotNil(t, a.responsePool)
	assert.IsType(t, &Request{}, a.requestPool.Get())
	assert.IsType(t, &Response{}, a.responsePool.Get())

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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodPost, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPut, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodPatch, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodDelete, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodConnect, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodOptions, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Matched [* /foobar]", rec.Body.String())

	req = httptest.NewRequest(http.MethodTrace, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar", rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	a.FILE("/foobar2", "nowhere")

	req = httptest.NewRequest(http.MethodGet, "/foobar2", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, http.StatusText(rec.Code), rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar2", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
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
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Foobar", rec.Body.String())

	req = httptest.NewRequest(
		http.MethodHead,
		path.Join("/foobar", filepath.Base(f.Name())),
		nil,
	)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/foobar/nowhere", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, http.StatusText(rec.Code), rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar/nowhere", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.String())

	a.FILES("/foobar2/", "")

	req = httptest.NewRequest(http.MethodGet, "/foobar2/air.go", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Body.String())

	req = httptest.NewRequest(http.MethodHead, "/foobar2/air.go", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
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

	a = New()
	a.Address = "-1:0"

	assert.Error(t, a.Serve())

	a = New()
	a.Address = ""

	assert.Error(t, a.Serve())

	a = New()
	a.Address = ":-1"

	assert.Error(t, a.Serve())

	a = New()
	a.Address = ""

	assert.Error(t, a.Serve())

	dir, err := ioutil.TempDir("", "air.TestAirServe")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a = New()
	a.DebugMode = true
	a.Address = "localhost:0"

	stdout, err := ioutil.TempFile(dir, "")
	assert.NoError(t, err)

	stdoutBackup := os.Stdout
	os.Stdout = stdout

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	os.Stdout = stdoutBackup

	assert.NoError(t, stdout.Close())

	b, err := ioutil.ReadFile(stdout.Name())
	assert.NoError(t, err)
	assert.Equal(
		t,
		fmt.Sprintf(
			"air: serving in debug mode\nair: listening on %v\n",
			a.Addresses(),
		),
		string(b),
	)

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "localhost:0"

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   a.Addresses()[0],
		},
		Host: "localhost",
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   a.Addresses()[0],
		},
		Host: "example.com",
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "localhost:0"

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, "tls_cert.pem"),
		nil,
		os.ModePerm,
	))

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, "tls_key.pem"),
		nil,
		os.ModePerm,
	))

	a.TLSCertFile = filepath.Join(dir, "tls_cert.pem")
	a.TLSKeyFile = filepath.Join(dir, "tls_key.pem")

	assert.Error(t, a.Serve())

	a = New()
	a.Address = "localhost:0"
	a.HTTPSEnforced = true
	a.HTTPSEnforcedPort = "0"
	a.ErrorLogger = log.New(ioutil.Discard, "", 0)

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, "tls_cert.pem"),
		[]byte(`
-----BEGIN CERTIFICATE-----
MIIFBTCCA+2gAwIBAgISA19vMeUvx/Tnt3mnfnbQKzIEMA0GCSqGSIb3DQEBCwUA
MEoxCzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MSMwIQYDVQQD
ExpMZXQncyBFbmNyeXB0IEF1dGhvcml0eSBYMzAeFw0xNzAxMjIwMzA3MDBaFw0x
NzA0MjIwMzA3MDBaMBQxEjAQBgNVBAMTCWFpcndmLm9yZzCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAMqIYMFjNRADYUbnQhfyIc77M0in8eWD4iVAEXcj
lKUz/vf/Hxm1TfE+LQampJF57JceT0hfqmDNzt5W+52aN1P+wbx7XHa4F+3DdY5h
MVfxm36Y1y4/OKAsNBpVlBhTtnFQJLIUO8c9mDs9VSX6DBCNSzAS/rSfnThlxDKN
qTaQVXIAN8+iqiiIrK4q0SSlW12jOzok/BXxbOtiTWXaLEVnzKUEsYTZMkdGiRZF
PyIJktIHY3eujG8c4tGr9KtX1b2ZvaaAIRcCOo0uhtJ18Sjb7IzQbz/Xba6LcqDL
3Q0HWO3UmIPxbzeTPgVSftdpC18ig9s7gLws38Rb1yifbskCAwEAAaOCAhkwggIV
MA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIw
DAYDVR0TAQH/BAIwADAdBgNVHQ4EFgQUJ3IaKlnvlxFNz5q5kBBJkUtcamAwHwYD
VR0jBBgwFoAUqEpqYwR93brm0Tm3pkVl7/Oo7KEwcAYIKwYBBQUHAQEEZDBiMC8G
CCsGAQUFBzABhiNodHRwOi8vb2NzcC5pbnQteDMubGV0c2VuY3J5cHQub3JnLzAv
BggrBgEFBQcwAoYjaHR0cDovL2NlcnQuaW50LXgzLmxldHNlbmNyeXB0Lm9yZy8w
IwYDVR0RBBwwGoIJYWlyd2Yub3Jngg13d3cuYWlyd2Yub3JnMIH+BgNVHSAEgfYw
gfMwCAYGZ4EMAQIBMIHmBgsrBgEEAYLfEwEBATCB1jAmBggrBgEFBQcCARYaaHR0
cDovL2Nwcy5sZXRzZW5jcnlwdC5vcmcwgasGCCsGAQUFBwICMIGeDIGbVGhpcyBD
ZXJ0aWZpY2F0ZSBtYXkgb25seSBiZSByZWxpZWQgdXBvbiBieSBSZWx5aW5nIFBh
cnRpZXMgYW5kIG9ubHkgaW4gYWNjb3JkYW5jZSB3aXRoIHRoZSBDZXJ0aWZpY2F0
ZSBQb2xpY3kgZm91bmQgYXQgaHR0cHM6Ly9sZXRzZW5jcnlwdC5vcmcvcmVwb3Np
dG9yeS8wDQYJKoZIhvcNAQELBQADggEBAEeZuWoMm5E9V/CQxQv0GBJEr3jl7e/O
Wauwl+sRLbQG9ajHlnKz46Af/oDoG4Z+e7iYRRZm9nIOLVCsp3Yp+h+GSjwm8yiP
fwAyaLfBKNbtEk0S/FNmqzr7jjxCyHhqoloHhzFAfHJyhlYlMUwQhbxM1U5GbejE
9ru76RTbdh3yb00HSXBMcc3woiaGWPc8FVaT8LGOweKIEH4kcYevC06m860ovHV/
s87+zaamZW4j8uWLGPxS4eD2Ulg+nbLKdnprbYEx5F943M1b7s05LJ+E7SnqKS3i
jiepPCVdRmlsROMoSfWQXFdfsTKEFAwOeIbIxfk7EgUIzrUgnnv0G7Q=
-----END CERTIFICATE-----
		`),
		os.ModePerm,
	))

	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, "tls_key.pem"),
		[]byte(`
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDKiGDBYzUQA2FG
50IX8iHO+zNIp/Hlg+IlQBF3I5SlM/73/x8ZtU3xPi0GpqSReeyXHk9IX6pgzc7e
VvudmjdT/sG8e1x2uBftw3WOYTFX8Zt+mNcuPzigLDQaVZQYU7ZxUCSyFDvHPZg7
PVUl+gwQjUswEv60n504ZcQyjak2kFVyADfPoqooiKyuKtEkpVtdozs6JPwV8Wzr
Yk1l2ixFZ8ylBLGE2TJHRokWRT8iCZLSB2N3roxvHOLRq/SrV9W9mb2mgCEXAjqN
LobSdfEo2+yM0G8/122ui3Kgy90NB1jt1JiD8W83kz4FUn7XaQtfIoPbO4C8LN/E
W9con27JAgMBAAECggEAFUx6QFwafHCejkJLpREFlSq9nepreeOAqMIwFANd4nGx
YoslziJO7AvJ2GU18UaNJuc9FzNYS43ZL3CeTVimcOLdpOCkPKfnfE2N00dNVR5H
Z+zS1D45yj5bzFkrldNX4Fq5QTD3iGBl3fT5O2EsW6FAQvH8bypJ8mBhXZ+gJ+id
4croKKwMsHGYSiLdCSVf6oGkytlQwggAl0B85KBCOR1ArMf2nrM9lf6yBLJRGo6f
qzIEAvDPNicW5BWGf2lwQTmawKMecStWXniu8VdjKoRO9IXDe2WQAdwC8LjAQwxZ
hQJbM6I8x0CExMmEthieUrX0VkblboOC/BQsUzNwAQKBgQDurZ07acp/P9icDIUN
l53OiCafYrlBceZCdykheDHgpg+TBVfO8GUMsXywYIMOw1RzmGqDWWrU7uaiXnMn
kL/LKFM9t/10vFrlt5F1cx45MJsknVDebfJGq+L6eHISx+7igTCyQ6JBD4sW2tcs
c6MYHgVsAHioqrkcjvHBUY8cSQKBgQDZOzhFg41h3U+cTgePGjzZpziWB1VO8ird
OJp8Hn8umUW8JfdYTalTvzs2CiNw0gOjGETMUmKKhS2YcGIol9j7elBOhT9mzxKf
NHEJRiV6+2SInESUfcLaXZZQKbMMiw2YZfV2ADf8n+Lb79tlbAtSEnMnvmlDI/1K
SASXbGS+gQKBgQDeh7JUBaOOFsnvXGDlNlokiJ5x9krBMN+9UnpfwT/HsyxMKCwh
PdMJDaYykBlBN27Sw+VzB3hqhT81XZhB6FxZnwRVQ+kk4MRi707IUYd5TM8pSR9v
8tRzfakHXCsHRa99MXRkkFiEDmjg6zK5OCt0vfDSLHJS17H1ZXUTh+ZFOQKBgFgX
1OUTyTUDu7ImTphwynZ1gtQMm0LNoCZgOv3UnDz4eTgoqVrM+7rzlP6ANAkfkcwF
HnlBe6azBV+JS7UshxjMbF67WI/Hr8SSTri1EqQB6K4huQoCyg8l3rwZfPu8NEI2
LsmwowO2jxgj9/P0Uc7xnnNim2tX3/LMq9gAZAaBAoGBALI4Y4/lBNfBRB0IIA+p
Edt9VRdifXbQE+q1JwyG9smGsumYuMCBGQFZp51Wa5/FD/NRqezRDP3myiRQzWiM
fNAWEfZaazKKFmOrC4WgM+Z8bKAyrDpmCu2iNvdS2JPYujiIX+f5kq7W0muF4JXZ
l7j2fuWjNfj9JfnXoP2SEgPG
-----END PRIVATE KEY-----
		`),
		os.ModePerm,
	))

	a.TLSCertFile = filepath.Join(dir, "tls_cert.pem")
	a.TLSKeyFile = filepath.Join(dir, "tls_key.pem")

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   a.Addresses()[0],
		},
		Host: "example.com",
	})
	assert.Error(t, err)
	assert.Nil(t, res)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   a.Addresses()[0],
		},
		Host: "localhost",
	})
	http.DefaultTransport.(*http.Transport).TLSClientConfig = nil
	assert.NoError(t, err)
	assert.NotNil(t, res)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   a.Addresses()[1],
		},
		Host: "localhost",
	})
	http.DefaultTransport.(*http.Transport).TLSClientConfig = nil
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.NoError(t, a.Close())

	c, err := tls.LoadX509KeyPair(
		filepath.Join(dir, "tls_cert.pem"),
		filepath.Join(dir, "tls_key.pem"),
	)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	a = New()
	a.Address = "localhost:0"
	a.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{c},
	}

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "-1:0"
	a.TLSCertFile = filepath.Join(dir, "tls_cert.pem")
	a.TLSKeyFile = filepath.Join(dir, "tls_key.pem")

	assert.Error(t, a.Serve())

	a = New()
	a.Address = "localhost:0"
	a.HTTPSEnforced = true
	a.HTTPSEnforcedPort = "-1"
	a.TLSCertFile = filepath.Join(dir, "tls_cert.pem")
	a.TLSKeyFile = filepath.Join(dir, "tls_key.pem")

	assert.Error(t, a.Serve())

	a = New()
	a.Address = "localhost:0"
	a.ACMEEnabled = true
	a.ACMECertRoot = dir
	a.ACMEHostWhitelist = []string{"localhost"}
	a.HTTPSEnforcedPort = "0"
	a.ErrorLogger = log.New(ioutil.Discard, "", 0)

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   a.Addresses()[0],
		},
		Host: "example.com",
	})
	assert.Error(t, err)
	assert.Nil(t, res)

	assert.NoError(t, a.Close())

	a = New()
	a.Address = "localhost:0"
	a.ACMEEnabled = true
	a.ACMECertRoot = dir
	a.ACMEHostWhitelist = []string{"localhost"}
	a.HTTPSEnforcedPort = "0"
	a.ErrorLogger = log.New(ioutil.Discard, "", 0)

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   a.Addresses()[0],
		},
		Host: "example.com",
	})
	assert.Error(t, err)
	assert.Nil(t, res)

	assert.NoError(t, a.Close())
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

	foo := ""
	a.AddShutdownJob(func() {
		foo = "bar"
	})

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Shutdown(context.Background()))
	assert.Equal(t, "bar", foo)
	assert.Len(t, a.shutdownJobs, 1)

	a = New()
	a.Address = "localhost:0"

	foo = ""
	a.AddShutdownJob(func() {
		time.Sleep(100 * time.Millisecond)
		foo = "bar"
	})

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.Error(t, context.Canceled, a.Shutdown(ctx))
	assert.Empty(t, foo)
	assert.Len(t, a.shutdownJobs, 1)
}

func TestAirAddShutdownJob(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	foo := ""
	id := a.AddShutdownJob(func() {
		foo = "bar"
	})

	assert.Equal(t, 0, id)

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Shutdown(context.Background()))
	assert.Equal(t, "bar", foo)
}

func TestAirRemoveShutdownJob(t *testing.T) {
	a := New()
	a.Address = "localhost:0"

	foo := ""
	id := a.AddShutdownJob(func() {
		foo = "bar"
	})

	assert.Equal(t, 0, id)

	a.RemoveShutdownJob(id)

	hijackOSStdout()

	go a.Serve()
	time.Sleep(100 * time.Millisecond)

	revertOSStdout()

	assert.NoError(t, a.Shutdown(context.Background()))
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
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, a.Addresses(), 0)
}

func TestAirServeHTTP(t *testing.T) {
	a := New()
	a.Pregases = []Gas{func(next Handler) Handler {
		return func(req *Request, res *Response) error {
			req.SetValue("EasterEgg", easterEgg)

			res.Defer(func() {
				res.WriteString("Defer")
			})

			if err := res.WriteString("Pregas - "); err != nil {
				return err
			}

			return next(req, res)
		}
	}}
	a.Gases = []Gas{func(next Handler) Handler {
		return func(req *Request, res *Response) error {
			if err := res.WriteString("Gas - "); err != nil {
				return err
			}

			return next(req, res)
		}
	}}

	a.GET("/hello/:Name", func(req *Request, res *Response) error {
		if req.Value("EasterEgg") != easterEgg {
			return errors.New("wrong easter egg")
		}

		return res.WriteString(
			"Hello, " + req.Param("Name").Value().String() + " - ",
		)
	})

	req := httptest.NewRequest(http.MethodGet, "/hello/Air", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "Pregas - Gas - Hello, Air - Defer", rec.Body.String())

	a = New()

	a.GET("/", func(req *Request, res *Response) error {
		return errors.New("handler error")
	})

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "Internal Server Error", rec.Body.String())

	a = New()
	a.DebugMode = true

	a.GET("/:Foo", func(req *Request, res *Response) error {
		return errors.New("handler error")
	})

	req = httptest.NewRequest(http.MethodGet, "/bar", nil)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "handler error", rec.Body.String())
}

func TestAirLogErrorf(t *testing.T) {
	a := New()

	buf := bytes.Buffer{}

	log.SetOutput(&buf)
	log.SetFlags(0)
	a.logErrorf("air: some error: %v", errors.New("foobar"))
	assert.Equal(t, "air: some error: foobar\n", buf.String())
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	buf.Reset()

	a.ErrorLogger = log.New(&buf, "", 0)
	a.logErrorf("air: some error: %v", errors.New("foobar"))
	assert.Equal(t, "air: some error: foobar\n", buf.String())
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
	assert.True(t, stringSliceContains([]string{"foo"}, "foo", false))
	assert.True(t, stringSliceContains([]string{"foo"}, "foo", true))
	assert.False(t, stringSliceContains([]string{"foo"}, "Foo", false))
	assert.True(t, stringSliceContains([]string{"foo"}, "FOO", true))
	assert.False(t, stringSliceContains([]string{"foo"}, "bar", false))
	assert.False(t, stringSliceContains([]string{"foo"}, "BAR", true))
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
	}
	res.SetHTTPResponseWriter(&responseWriter{
		r:  res,
		rw: rec,
	})

	req.res = res
	res.req = req

	return req, res, rec
}

var osStdout = os.Stdout

func hijackOSStdout() {
	os.Stdout, _ = ioutil.TempFile("", "air.FakeStdout")
}

func revertOSStdout() {
	if os.Stdout != osStdout {
		os.Remove(os.Stdout.Name())
	}

	os.Stdout = osStdout
}

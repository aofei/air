package air

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAirNew(t *testing.T) {
	a := New()

	assert.Equal(t, 0, len(a.pregases))
	assert.Equal(t, 0, len(a.gases))
	assert.Equal(t, 0, a.paramCap)
	assert.NotNil(t, a.contextPool)
	assert.NotNil(t, a.server)
	assert.NotNil(t, a.router)

	assert.NotNil(t, a.Config)
	assert.NotNil(t, a.Logger)
	assert.NotNil(t, a.Binder)
	assert.NotNil(t, a.Minifier)
	assert.NotNil(t, a.Renderer)
	assert.NotNil(t, a.Coffer)
	assert.NotNil(t, a.HTTPErrorHandler)
}

func TestAirPrecontain(t *testing.T) {
	a := New()
	a.server = newServer(a)
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()

	pregas := WrapGas(func(c *Context) error { return c.String("pregas") })

	a.Precontain(pregas)
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, "pregas", rec.Body.String())
}

func TestAirContain(t *testing.T) {
	a := New()
	a.server = newServer(a)
	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()

	gas := WrapGas(func(c *Context) error { return c.String("gas") })

	a.Contain(gas)
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, "gas", rec.Body.String())
}

func TestAirMethods(t *testing.T) {
	a := New()
	a.server = newServer(a)
	path := "/methods"
	req, _ := http.NewRequest(GET, path, nil)
	rec := httptest.NewRecorder()

	a.GET(path, func(c *Context) error { return c.String(GET) })
	a.POST(path, func(c *Context) error { return c.String(POST) })
	a.PUT(path, func(c *Context) error { return c.String(PUT) })
	a.DELETE(path, func(c *Context) error { return c.String(DELETE) })

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, GET, rec.Body.String())

	req.Method = POST
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, POST, rec.Body.String())

	req.Method = PUT
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, PUT, rec.Body.String())

	req.Method = DELETE
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, DELETE, rec.Body.String())
}

func TestAirStatic(t *testing.T) {
	a := New()
	a.server = newServer(a)
	prefix := "/air"
	fn := "air.go"
	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest(GET, prefix+"/"+fn, nil)
	rec := httptest.NewRecorder()

	a.Static(prefix, ".")

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())

	fn = "air_test.go"
	b, _ = ioutil.ReadFile(fn)
	req, _ = http.NewRequest(GET, prefix+"/"+fn, nil)
	rec = httptest.NewRecorder()
	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestAirFile(t *testing.T) {
	a := New()
	a.server = newServer(a)
	path := "/air"
	fn := "air.go"
	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest(GET, path, nil)
	rec := httptest.NewRecorder()

	a.File(path, fn)

	a.server.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestAirURL(t *testing.T) {
	a := New()
	h := func(c *Context) error { return c.NoContent() }
	a.GET("/:first/:second", h)
	assert.Equal(t, "/foo/bar", a.URL(h, "foo", "bar"))
}

func TestAirServe(t *testing.T) {
	a := New()
	ok := make(chan struct{})

	go func() {
		close(ok)
		a.Serve()
	}()

	<-ok
	assert.NoError(t, a.Close())
}

type failingMinifier struct{}

func (*failingMinifier) Init() error {
	return errors.New("failingMinifier")
}

func (*failingMinifier) Minify(mimeType string, b []byte) ([]byte, error) {
	return nil, nil
}

type failingRenderer struct{}

func (*failingRenderer) SetTemplateFunc(name string, f interface{}) {}

func (*failingRenderer) Init() error {
	return errors.New("failingRenderer")
}

func (*failingRenderer) Render(w io.Writer, templateName string, data Map) error {
	return nil
}

type failingCoffer struct{}

func (*failingCoffer) Init() error {
	return errors.New("failingCoffer")
}

func (*failingCoffer) Asset(name string) *Asset {
	return nil
}

func (*failingCoffer) SetAsset(a *Asset) {}

func TestAirServeParseTemplatesError(t *testing.T) {
	a := New()
	buf := &bytes.Buffer{}
	ok := make(chan struct{})

	a.Logger.SetOutput(buf)
	a.Config.LoggerEnabled = true
	a.Minifier = &failingMinifier{}
	a.Renderer = &failingRenderer{}
	a.Coffer = &failingCoffer{}

	go func() {
		close(ok)
		a.Serve()
	}()

	<-ok
	time.Sleep(time.Millisecond) // Wait for logger
	assert.Contains(t, buf.String(), "failingMinifier")
	assert.Contains(t, buf.String(), "failingRenderer")
	assert.Contains(t, buf.String(), "failingCoffer")
	assert.NoError(t, a.Close())
}

func TestAirServeTLS(t *testing.T) {
	cert := `
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
`
	key := `
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
`

	c, _ := os.Create("cert.pem")
	defer func() {
		c.Close()
		os.Remove(c.Name())
	}()
	c.WriteString(cert)

	k, _ := os.Create("key.pem")
	defer func() {
		k.Close()
		os.Remove(k.Name())
	}()
	k.WriteString(key)

	a := New()
	ok := make(chan struct{})

	a.Config.TLSCertFile = c.Name()
	a.Config.TLSKeyFile = k.Name()

	go func() {
		close(ok)
		a.Serve()
	}()

	<-ok
	assert.NoError(t, a.Shutdown(NewContext(a)))
}

func TestAirServeDebugMode(t *testing.T) {
	a := New()
	buf := &bytes.Buffer{}
	ok := make(chan struct{})

	a.Config.DebugMode = true
	a.Logger.SetOutput(buf)

	go func() {
		close(ok)
		a.Serve()
	}()

	<-ok
	time.Sleep(time.Millisecond) // Wait for logger
	assert.Equal(t, true, a.Config.LoggerEnabled)
	assert.NotEmpty(t, buf.String())
	assert.NoError(t, a.Close())
}

type httpHandler struct{}

func (*httpHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func TestAirWrapHandler(t *testing.T) {
	a := New()
	c := a.contextPool.Get().(*Context)
	h := WrapHandler(&httpHandler{})

	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()

	c.feed(req, rec)
	h(c)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAirWrapGasError(t *testing.T) {
	g := WrapGas(func(*Context) error { return ErrInternalServerError })
	h := g(func(*Context) error { return nil })
	assert.Equal(t, ErrInternalServerError, h(nil))
}

func TestAirDefaultHTTPErrorHandler(t *testing.T) {
	a := New()
	a.Config.DebugMode = true
	c := a.contextPool.Get().(*Context)

	req, _ := http.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()

	c.feed(req, rec)

	he := NewHTTPError(http.StatusInternalServerError, "error")
	DefaultHTTPErrorHandler(he, c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, he.Error(), rec.Body.String())

	c = a.contextPool.Get().(*Context)

	req, _ = http.NewRequest(GET, "/", nil)
	rec = httptest.NewRecorder()

	c.feed(req, rec)

	err := errors.New("error")
	DefaultHTTPErrorHandler(err, c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, err.Error(), rec.Body.String())
}

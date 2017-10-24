package air

import (
	"bytes"
	"errors"
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

	assert.NotNil(t, a.Logger)
	assert.NotNil(t, a.server)
	assert.NotNil(t, a.router)
	assert.NotNil(t, a.binder)
	assert.NotNil(t, a.minifier)
	assert.NotNil(t, a.renderer)
	assert.NotNil(t, a.coffer)
}

func TestAirMethods(t *testing.T) {
	a := New()
	s := a.server
	path := "/methods"
	req, _ := http.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()

	a.GET(path, func(req *Request, res *Response) error {
		return res.String("GET")
	})
	a.HEAD(path, func(req *Request, res *Response) error {
		return res.String("HEAD")
	})
	a.POST(path, func(req *Request, res *Response) error {
		return res.String("POST")
	})
	a.PUT(path, func(req *Request, res *Response) error {
		return res.String("PUT")
	})
	a.PATCH(path, func(req *Request, res *Response) error {
		return res.String("PATCH")
	})
	a.DELETE(path, func(req *Request, res *Response) error {
		return res.String("DELETE")
	})
	a.CONNECT(path, func(req *Request, res *Response) error {
		return res.String("CONNECT")
	})
	a.OPTIONS(path, func(req *Request, res *Response) error {
		return res.String("OPTIONS")
	})
	a.TRACE(path, func(req *Request, res *Response) error {
		return res.String("TRACE")
	})

	s.ServeHTTP(rec, req)
	assert.Equal(t, "GET", rec.Body.String())

	req.Method = "GET"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "GET", rec.Body.String())

	req.Method = "POST"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "POST", rec.Body.String())

	req.Method = "PUT"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "PUT", rec.Body.String())

	req.Method = "PATCH"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "PATCH", rec.Body.String())

	req.Method = "DELETE"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "DELETE", rec.Body.String())

	req.Method = "CONNECT"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "CONNECT", rec.Body.String())

	req.Method = "OPTIONS"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "OPTIONS", rec.Body.String())

	req.Method = "TRACE"
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, "TRACE", rec.Body.String())
}

func TestAirSTATIC(t *testing.T) {
	a := New()
	s := a.server
	prefix := "/air"
	fn := "air.go"
	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest("GET", prefix+"/"+fn, nil)
	rec := httptest.NewRecorder()

	a.STATIC(prefix, ".")

	s.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())

	fn = "air_test.go"
	b, _ = ioutil.ReadFile(fn)
	req, _ = http.NewRequest("GET", prefix+"/"+fn, nil)
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
}

func TestAirFILE(t *testing.T) {
	a := New()
	s := a.server
	path := "/air"
	fn := "air.go"
	b, _ := ioutil.ReadFile(fn)
	req, _ := http.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()

	a.FILE(path, fn)

	s.ServeHTTP(rec, req)
	assert.Equal(t, b, rec.Body.Bytes())
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

	a.TLSCertFile = c.Name()
	a.TLSKeyFile = k.Name()

	go func() {
		close(ok)
		a.Serve()
	}()

	<-ok
	assert.NoError(t, a.Shutdown(-1))
}

func TestAirServeDebugMode(t *testing.T) {
	a := New()
	buf := &bytes.Buffer{}
	ok := make(chan struct{})

	a.DebugMode = true
	a.Logger.Output = buf

	go func() {
		close(ok)
		a.Serve()
	}()

	<-ok
	time.Sleep(time.Millisecond) // Wait for logger
	assert.Equal(t, true, a.LoggerEnabled)
	assert.NotEmpty(t, buf.String())
	assert.NoError(t, a.Close())
}

type httpHandler struct{}

func (*httpHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(200)
}

func TestAirWrapGasError(t *testing.T) {
	g := WrapGas(func(*Request, *Response) error {
		return errors.New("gas error")
	})
	h := g(func(*Request, *Response) error {
		return nil
	})
	assert.Equal(t, "gas error", h(nil, nil).Error())
}

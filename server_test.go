package air

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	a := New()
	s := a.server

	assert.NotNil(t, s)
	assert.NotNil(t, s.a)
	assert.NotNil(t, s.server)
	assert.Nil(t, s.allowedPROXYProtocolRelayerIPNets)
	assert.NotNil(t, s.requestPool)
	assert.NotNil(t, s.responsePool)
	assert.IsType(t, &Request{}, s.requestPool.Get())
	assert.IsType(t, &Response{}, s.responsePool.Get())
}

func TestServerServe(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	s := a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	assert.NoError(t, s.close())

	a = New()
	a.Address = "localhost:8080"
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolRelayerIPWhitelist = []string{
		"0.0.0.0",
		"::",
		"127.0.0.1",
		"127.0.0.1/32",
		"::1",
		"::1/128",
	}

	s = a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, s.allowedPROXYProtocolRelayerIPNets, 6)

	assert.NoError(t, s.close())

	a = New()
	a.Address = ":-1"

	s = a.server

	assert.Error(t, s.serve())

	a = New()
	a.Address = ""

	s = a.server

	assert.Error(t, s.serve())

	dir, err := ioutil.TempDir("", "air.TestServerServe")
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer os.RemoveAll(dir)

	a = New()
	a.DebugMode = true
	a.Address = "localhost:8080"

	s = a.server

	stdout, err := ioutil.TempFile(dir, "")
	assert.NoError(t, err)

	stdoutBackup := os.Stdout
	os.Stdout = stdout

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	os.Stdout = stdoutBackup

	assert.NoError(t, stdout.Close())

	b, err := ioutil.ReadFile(stdout.Name())
	assert.NoError(t, err)
	assert.Equal(t, "air: serving in debug mode\n", string(b))

	assert.NoError(t, s.close())

	a = New()
	a.Address = "localhost:8080"

	s = a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:8080",
		},
		Host: "localhost",
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:8080",
		},
		Host: "example.com",
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.NoError(t, s.close())

	a = New()
	a.Address = "localhost:1443"

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

	s = a.server

	assert.Error(t, s.serve())

	a = New()
	a.Address = "localhost:1443"
	a.HTTPSEnforced = true
	a.HTTPSEnforcedPort = "8080"
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

	s = a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "localhost:1443",
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
			Host:   "localhost:1443",
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
			Host:   "localhost:8080",
		},
		Host: "localhost",
	})
	http.DefaultTransport.(*http.Transport).TLSClientConfig = nil
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.NoError(t, s.close())

	a = New()
	a.Address = ":-1"
	a.TLSCertFile = filepath.Join(dir, "tls_cert.pem")
	a.TLSKeyFile = filepath.Join(dir, "tls_key.pem")

	s = a.server

	assert.Error(t, s.serve())

	a = New()
	a.Address = "localhost:1443"
	a.HTTPSEnforced = true
	a.HTTPSEnforcedPort = "-1"
	a.TLSCertFile = filepath.Join(dir, "tls_cert.pem")
	a.TLSKeyFile = filepath.Join(dir, "tls_key.pem")

	s = a.server

	assert.Error(t, s.serve())

	a = New()
	a.Address = "localhost:1443"
	a.ACMEEnabled = true
	a.ACMECertRoot = dir
	a.ACMEHostWhitelist = []string{"localhost"}
	a.HTTPSEnforcedPort = "8080"
	a.ErrorLogger = log.New(ioutil.Discard, "", 0)

	s = a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "localhost:1443",
		},
		Host: "example.com",
	})
	assert.Error(t, err)
	assert.Nil(t, res)

	assert.NoError(t, s.close())

	a = New()
	a.Address = "localhost:1443"
	a.ACMEEnabled = true
	a.ACMECertRoot = dir
	a.ACMEHostWhitelist = []string{"localhost"}
	a.HTTPSEnforcedPort = "8080"
	a.ErrorLogger = log.New(ioutil.Discard, "", 0)

	s = a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	res, err = http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "localhost:1443",
		},
		Host: "example.com",
	})
	assert.Error(t, err)
	assert.Nil(t, res)

	assert.NoError(t, s.close())
}

func TestServerClose(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	s := a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	assert.NoError(t, s.close())
}

func TestServerAllowedPROXYProtocolRelayerIP(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"
	a.PROXYProtocolEnabled = true

	s := a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	ra, err := net.ResolveTCPAddr("tcp", "127.0.0.1:80")
	assert.NotNil(t, ra)
	assert.NoError(t, err)

	allowed, err := s.allowedPROXYProtocolRelayerIP(ra)
	assert.True(t, allowed)
	assert.NoError(t, err)

	ra, err = net.ResolveTCPAddr("tcp", "127.0.0.2:80")
	assert.NotNil(t, ra)
	assert.NoError(t, err)

	allowed, err = s.allowedPROXYProtocolRelayerIP(ra)
	assert.True(t, allowed)
	assert.NoError(t, err)

	assert.NoError(t, s.close())

	a = New()
	a.Address = "localhost:8080"
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolRelayerIPWhitelist = []string{"127.0.0.1"}

	s = a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	ra, err = net.ResolveTCPAddr("tcp", "127.0.0.1:80")
	assert.NotNil(t, ra)
	assert.NoError(t, err)

	allowed, err = s.allowedPROXYProtocolRelayerIP(ra)
	assert.True(t, allowed)
	assert.Nil(t, err)

	ra, err = net.ResolveTCPAddr("tcp", "127.0.0.2:80")
	assert.NotNil(t, allowed)
	assert.NoError(t, err)

	allowed, err = s.allowedPROXYProtocolRelayerIP(ra)
	assert.False(t, allowed)
	assert.NoError(t, err)

	assert.NoError(t, s.close())
}

func TestServerShutdown(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	s := a.server

	go s.serve()
	time.Sleep(100 * time.Millisecond)

	assert.NoError(t, s.shutdown(context.Background()))
}

func TestServerServeHTTP(t *testing.T) {
	a := New()
	a.Pregases = []Gas{func(next Handler) Handler {
		return func(req *Request, res *Response) error {
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
		return res.WriteString(
			"Hello, " + req.Param("Name").Value().String() + " - ",
		)
	})

	s := a.server

	req := httptest.NewRequest(http.MethodGet, "/hello/Air", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "Pregas - Gas - Hello, Air - Defer", rec.Body.String())

	a = New()

	a.GET("/", func(req *Request, res *Response) error {
		return errors.New("Handler error")
	})

	s = a.server

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)

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
		return errors.New("Handler error")
	})

	s = a.server

	req = httptest.NewRequest(http.MethodGet, "/bar", nil)
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(
		t,
		"text/plain; charset=utf-8",
		rec.HeaderMap.Get("Content-Type"),
	)
	assert.Equal(t, "Handler error", rec.Body.String())
}

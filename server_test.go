package air

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	assert.NotNil(t, theServer)
	assert.NotNil(t, theServer.server)
}

func TestServerServe(t *testing.T) {
	assert.False(t, DebugMode)
	assert.Equal(t, LoggerLowestLevel, LoggerLevelDebug)

	DebugMode = true
	LoggerLowestLevel = LoggerLevelOff

	buf := bytes.Buffer{}
	LoggerOutput = &buf

	go func() {
		assert.Error(t, http.ErrServerClosed, theServer.serve())
	}()

	time.Sleep(100 * time.Millisecond)

	ss := theServer.server
	assert.Equal(t, Address, ss.Addr)
	assert.Equal(t, theServer, ss.Handler)
	assert.Equal(t, IdleTimeout, ss.IdleTimeout)
	assert.Equal(t, LoggerLowestLevel, LoggerLevelDebug)

	m := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "air: serving in debug mode", m["message"])

	assert.NoError(t, theServer.close())

	DebugMode = false
	LoggerLowestLevel = LoggerLevelDebug
	LoggerOutput = os.Stdout
	theServer.server = &http.Server{}
}

func TestServerServeTLS(t *testing.T) {
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

	TLSCertFile = "foobar.crt"
	TLSKeyFile = "foobar.key"

	certFile, _ := os.Create(TLSCertFile)
	defer func() {
		certFile.Close()
		os.Remove(certFile.Name())
	}()
	certFile.WriteString(cert)

	keyFile, _ := os.Create(TLSKeyFile)
	defer func() {
		keyFile.Close()
		os.Remove(keyFile.Name())
	}()
	keyFile.WriteString(key)

	go func() {
		assert.Error(t, http.ErrServerClosed, theServer.serve())
	}()

	time.Sleep(100 * time.Millisecond)

	assert.NoError(t, theServer.shutdown(0))
	assert.NoError(t, theServer.shutdown(1))

	theServer.server = &http.Server{}
}

func TestServerSeveHTTP(t *testing.T) {
	LoggerLowestLevel = LoggerLevelOff

	buf := bytes.Buffer{}

	Pregases = []Gas{
		func(next Handler) Handler {
			return func(req *Request, res *Response) error {
				buf.WriteString("Pregas\n")
				return next(req, res)
			}
		},
	}

	Gases = []Gas{
		func(next Handler) Handler {
			return func(req *Request, res *Response) error {
				buf.WriteString("Gas\n")
				return next(req, res)
			}
		},
	}

	GET(
		"/",
		func(req *Request, res *Response) error {
			buf.WriteString("Handler")
			return errors.New("handler error")
		},
		func(next Handler) Handler {
			return func(req *Request, res *Response) error {
				buf.WriteString("Route gas\n")
				return next(req, res)
			}
		},
	)

	go func() {
		assert.Error(t, http.ErrServerClosed, theServer.serve())
	}()

	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	theServer.ServeHTTP(rec, req)

	assert.Equal(t, "Pregas\nGas\nRoute gas\nHandler", buf.String())
	assert.Equal(t, 500, rec.Code)
	assert.Equal(t, "internal server error", rec.Body.String())

	theServer.server = &http.Server{}

	LoggerLowestLevel = LoggerLevelDebug
}

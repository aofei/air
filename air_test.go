package air

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	assert.NotNil(t, a.Renderer)
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

	a.Static(prefix, "./")

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
	go a.Serve()
}

func TestAirServeTLS(t *testing.T) {
	cert := `
-----BEGIN CERTIFICATE-----
MIIEkjCCA3qgAwIBAgIQCgFBQgAAAVOFc2oLheynCDANBgkqhkiG9w0BAQsFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTE2MDMxNzE2NDA0NloXDTIxMDMxNzE2NDA0Nlow
SjELMAkGA1UEBhMCVVMxFjAUBgNVBAoTDUxldCdzIEVuY3J5cHQxIzAhBgNVBAMT
GkxldCdzIEVuY3J5cHQgQXV0aG9yaXR5IFgzMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAnNMM8FrlLke3cl03g7NoYzDq1zUmGSXhvb418XCSL7e4S0EF
q6meNQhY7LEqxGiHC6PjdeTm86dicbp5gWAf15Gan/PQeGdxyGkOlZHP/uaZ6WA8
SMx+yk13EiSdRxta67nsHjcAHJyse6cF6s5K671B5TaYucv9bTyWaN8jKkKQDIZ0
Z8h/pZq4UmEUEz9l6YKHy9v6Dlb2honzhT+Xhq+w3Brvaw2VFn3EK6BlspkENnWA
a6xK8xuQSXgvopZPKiAlKQTGdMDQMc2PMTiVFrqoM7hD8bEfwzB/onkxEz0tNvjj
/PIzark5McWvxI0NHWQWM6r6hCm21AvA2H3DkwIDAQABo4IBfTCCAXkwEgYDVR0T
AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwfwYIKwYBBQUHAQEEczBxMDIG
CCsGAQUFBzABhiZodHRwOi8vaXNyZy50cnVzdGlkLm9jc3AuaWRlbnRydXN0LmNv
bTA7BggrBgEFBQcwAoYvaHR0cDovL2FwcHMuaWRlbnRydXN0LmNvbS9yb290cy9k
c3Ryb290Y2F4My5wN2MwHwYDVR0jBBgwFoAUxKexpHsscfrb4UuQdf/EFWCFiRAw
VAYDVR0gBE0wSzAIBgZngQwBAgEwPwYLKwYBBAGC3xMBAQEwMDAuBggrBgEFBQcC
ARYiaHR0cDovL2Nwcy5yb290LXgxLmxldHNlbmNyeXB0Lm9yZzA8BgNVHR8ENTAz
MDGgL6AthitodHRwOi8vY3JsLmlkZW50cnVzdC5jb20vRFNUUk9PVENBWDNDUkwu
Y3JsMB0GA1UdDgQWBBSoSmpjBH3duubRObemRWXv86jsoTANBgkqhkiG9w0BAQsF
AAOCAQEA3TPXEfNjWDjdGBX7CVW+dla5cEilaUcne8IkCJLxWh9KEik3JHRRHGJo
uM2VcGfl96S8TihRzZvoroed6ti6WqEBmtzw3Wodatg+VyOeph4EYpr/1wXKtx8/
wApIvJSwtmVi4MFU5aMqrSDE6ea73Mj2tcMyo5jMd6jmeWUHK8so/joWUoHOUgwu
X4Po1QYz+3dszkDqMp4fklxBwXRsW10KXzPMTZ+sOPAveyxindmjkW8lGy+QsRlG
PfZ+G6Z6h7mjem0Y+iWlkYcV4PIWL1iwBi8saCbGS5jN2p8M+X+Q7UNKEkROb3N6
KOqkqm57TH2H3eDJAkSnh6/DNFu0Qg==
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

	c, err := os.Create("cert.pem")
	if err != nil {
		panic(err)
	}
	defer func() {
		c.Close()
		os.Remove(c.Name())
	}()

	c.WriteString(cert)

	k, err := os.Create("key.pem")
	if err != nil {
		panic(err)
	}
	defer func() {
		k.Close()
		os.Remove(k.Name())
	}()

	k.WriteString(key)

	a := New()

	a.Config.TLSCertFile = c.Name()
	a.Config.TLSKeyFile = k.Name()

	go a.Serve()
}

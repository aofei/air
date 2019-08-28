package air

import (
	"io"
	"log"
	"net/http"
	"runtime"
)

type route struct {
	method string
	path   string
}

var nullLogger *log.Logger
var loadTestHandler = false

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}
func httpHandlerFunc(w http.ResponseWriter, r *http.Request) {}

func httpHandlerFuncTest(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, r.RequestURI)
}
func (m *mockResponseWriter) WriteHeader(int) {}

// func main() {
// 	fmt.Println("Usage: go test -bench=.")
// 	os.Exit(1)
// }
func airHandler(req *Request, res *Response) error {
	return nil
}

func airHandlerWrite(req *Request, res *Response) error {
	// io.WriteString(res, s)
	return res.WriteString(req.Param("name").Value().String())
}
func airHandlerTest(req *Request, res *Response) error {
	return res.WriteString(req.Path)
}
func init() {
	runtime.GOMAXPROCS(1)

	// makes logging 'webscale' (ignores them)
	log.SetOutput(new(mockResponseWriter))
	nullLogger = log.New(new(mockResponseWriter), "", 0)
}

// // func (a *air.Air) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// // 	// a := air.New()
// // 	a.server.ServeHTTP(r, w)
// // }
func loadAir(routes []route) *Air {
	h := airHandler
	// if loadTestHandler {
	// 	h = airHandlerTest
	// }
	app := New()
	for _, r := range routes {
		switch r.method {
		case "GET":
			app.GET(r.path, h)
		case "POST":
			app.POST(r.path, h)
		case "PUT":
			app.PUT(r.path, h)
		case "PATCH":
			app.PATCH(r.path, h)
		case "DELETE":
			app.DELETE(r.path, h)
		default:
			panic("Unknow HTTP method: " + r.method)
		}
	}

	return app
}
func loadAirSingle(method, path string, h Handler) *Air {

	app := New()
	switch method {
	case "GET":
		app.GET(path, h)
	case "POST":
		app.POST(path, h)
	case "PUT":
		app.PUT(path, h)
	case "PATCH":
		app.PATCH(path, h)
	case "DELETE":
		app.DELETE(path, h)
	default:
		panic("Unknow HTTP method: " + method)
	}

	return app
}

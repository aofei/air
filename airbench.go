package air

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"time"
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
	var msg struct {
		Name string `json:"user"`
	}
	msg.Name = "Hello"
	return res.WriteJSON(msg)
}

func airHandlerWrite(req *Request, res *Response) error {
	var msg struct {
		Name string `json:"user"`
	}
	msg.Name = "Hello"
	return res.WriteJSON(msg)
}
func airHandlerTest(req *Request, res *Response) error {
	return res.WriteString(req.Path)
}
func airMiddleware(next Handler) Handler {
	return func(req *Request, res *Response) error {
		start := time.Now()
		err := next(req, res)
		responseTime := time.Since(start)

		// Write it to the log
		fmt.Println(responseTime)

		// Make sure to pass the error back!
		return err
	}
}
func init() {
	runtime.GOMAXPROCS(1)

	// makes logging 'webscale' (ignores them)
	log.SetOutput(new(mockResponseWriter))
	nullLogger = log.New(new(mockResponseWriter), "", 0)
}
func loadAirSingle(method, path string, h Handler) *Air {

	app := New()
	switch method {
	case "GET":
		app.GET(path, h, airMiddleware)
	case "POST":
		app.POST(path, h, airMiddleware)
	case "PUT":
		app.PUT(path, h, airMiddleware)
	case "PATCH":
		app.PATCH(path, h, airMiddleware)
	case "DELETE":
		app.DELETE(path, h, airMiddleware)
	default:
		panic("Unknow HTTP method: " + method)
	}

	return app
}

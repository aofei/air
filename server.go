package air

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// server is an HTTP server.
type server struct {
	server *http.Server
}

// theServer is the singleton of the `server`.
var theServer = &server{
	server: &http.Server{},
}

// serve starts the s.
func (s *server) serve() error {
	s.server.Addr = Address
	s.server.Handler = s
	s.server.ReadTimeout = ReadTimeout
	s.server.ReadHeaderTimeout = ReadHeaderTimeout
	s.server.WriteTimeout = WriteTimeout
	s.server.IdleTimeout = IdleTimeout
	s.server.MaxHeaderBytes = MaxHeaderBytes

	if DebugMode {
		LoggerEnabled = true
		INFO("serving in debug mode")
	}

	if TLSCertFile != "" && TLSKeyFile != "" {
		if HTTPSEnforced && (Address == "" ||
			strings.HasSuffix(strings.ToLower(Address), ":https") ||
			strings.HasSuffix(Address, ":443")) {
			a, _, err := net.SplitHostPort(Address)
			if err != nil {
				a = Address
			}
			var h http.HandlerFunc
			h = func(rw http.ResponseWriter, r *http.Request) {
				host, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					host = r.Host
				}
				url := "https://" + host + r.RequestURI
				http.Redirect(rw, r, url, 301)
			}
			go func() {
				if err = http.ListenAndServe(a, h); err != nil {
					FATAL(err)
				}
			}()
		}
		return s.server.ListenAndServeTLS(TLSCertFile, TLSKeyFile)
	}

	return s.server.ListenAndServe()
}

// close closes the s immediately.
func (s *server) close() error {
	return s.server.Close()
}

// shutdown gracefully shuts down the s without interrupting any active
// connections until timeout. It waits indefinitely for connections to return to
// idle and then shut down when the timeout is less than or equal to zero.
func (s *server) shutdown(timeout time.Duration) error {
	if timeout <= 0 {
		return s.server.Shutdown(context.Background())
	}

	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.server.Shutdown(c)
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Request

	req := &Request{
		Method: r.Method,
		URL: &URL{
			Scheme: "http",
			Host:   r.Host,
			Path:   r.URL.EscapedPath(),
			Query:  r.URL.RawQuery,
		},
		Proto:         "HTTP/" + strconv.Itoa(r.ProtoMajor),
		Headers:       make(map[string]string, len(r.Header)),
		Body:          r.Body,
		ContentLength: r.ContentLength,
		Cookies:       make([]*Cookie, 0, len(r.Header["Cookie"])),
		Params:        make(map[string]string, theRouter.maxParams),
		Files:         map[string]io.ReadSeeker{},
		RemoteAddr:    r.RemoteAddr,
		Values:        map[string]interface{}{},
	}

	if r.TLS != nil {
		req.URL.Scheme = "https"
	}

	if r.ProtoMajor < 2 {
		req.Proto += "." + strconv.Itoa(r.ProtoMinor)
	}

	for k, v := range r.Header {
		if len(v) > 0 {
			req.Headers[k] = strings.Join(v, ", ")
		}
	}

	for _, line := range r.Header["Cookie"] {
		parts := strings.Split(strings.TrimSpace(line), ";")
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		for i := 0; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}
			n, v := parts[i], ""
			if i := strings.Index(n, "="); i >= 0 {
				n, v = n[:i], n[i+1:]
			}
			if !validCookieName(n) {
				continue
			}
			if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}
			if !validCookieValue(v) {
				continue
			}
			req.Cookies = append(req.Cookies, &Cookie{
				Name:  n,
				Value: v,
			})
		}
	}

	if r.Form == nil || r.MultipartForm == nil {
		r.ParseMultipartForm(32 << 20)
	}

	for k, v := range r.Form {
		if len(v) > 0 {
			req.Params[k] = v[0]
		}
	}

	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.File {
			if len(v) > 0 {
				if f, err := v[0].Open(); err == nil {
					req.Files[k] = f
				}
			}
		}
	}

	// Response

	res := &Response{
		StatusCode: 200,
		Headers:    map[string]string{},

		request:     req,
		httpRequest: r,
		writer:      rw,
	}

	// Gases

	h := func(req *Request, res *Response) error {
		h := theRouter.route(req)
		for i := len(Gases) - 1; i >= 0; i-- {
			h = Gases[i](h)
		}
		return h(req, res)
	}

	// Pregases

	for i := len(Pregases) - 1; i >= 0; i-- {
		h = Pregases[i](h)
	}

	// Execute chain

	if err := h(req, res); err != nil {
		ErrorHandler(err, req, res)
	}
}

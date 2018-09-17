package air

import (
	"context"
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
		LoggerLowestLevel = LoggerLevelDebug
		DEBUG("air: serving in debug mode")
	}

	if TLSCertFile != "" && TLSKeyFile != "" {
		if HTTPSEnforced && (Address == "" ||
			strings.HasSuffix(strings.ToLower(Address), ":https") ||
			strings.HasSuffix(Address, ":443")) {
			go func() {
				a, _, err := net.SplitHostPort(Address)
				if err != nil {
					a = Address
				}

				if err := http.ListenAndServe(
					a,
					http.HandlerFunc(func(
						rw http.ResponseWriter,
						r *http.Request,
					) {
						h, _, err := net.SplitHostPort(
							r.Host,
						)
						if err != nil {
							h = r.Host
						}

						http.Redirect(
							rw,
							r,
							"https://"+
								h+
								r.RequestURI,
							301,
						)
					}),
				); err != nil {
					ERROR(
						"air: http2https handler error",
						map[string]interface{}{
							"error": err.Error(),
						},
					)
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
		Cookies:       map[string]*Cookie{},
		Params:        map[string]*RequestParamValue{},
		RemoteAddr:    r.RemoteAddr,
		Values:        map[string]interface{}{},

		httpRequest: r,
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

	cIPStr, _, _ := net.SplitHostPort(req.RemoteAddr)
	if f := req.Headers["Forwarded"]; f != "" { // See RFC 7239
		for _, p := range strings.Split(strings.Split(f, ",")[0], ";") {
			p := strings.TrimSpace(p)
			if strings.HasPrefix(p, "for=") {
				cIPStr = strings.TrimPrefix(p[4:], "\"[")
				cIPStr = strings.TrimSuffix(cIPStr, "]\"")
				break
			}
		}
	} else if xff := req.Headers["X-Forwarded-For"]; xff != "" {
		cIPStr = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	req.ClientIP = net.ParseIP(cIPStr)

	theI18n.localize(req)

	// Response

	res := &Response{
		StatusCode: 200,
		Headers:    map[string]string{},
		Cookies:    map[string]*Cookie{},

		request: req,
		writer:  rw,
	}

	// Chain gases

	h := func(req *Request, res *Response) error {
		rh := theRouter.route(req)
		h := func(req *Request, res *Response) error {
			if err := rh(req, res); err != nil {
				return err
			} else if !res.Written {
				return res.Write(nil)
			}

			return nil
		}

		req.ParseCookies()
		req.ParseParams()

		for i := len(Gases) - 1; i >= 0; i-- {
			h = Gases[i](h)
		}

		return h(req, res)
	}

	// Chain pregases

	for i := len(Pregases) - 1; i >= 0; i-- {
		h = Pregases[i](h)
	}

	// Execute chain

	if err := h(req, res); err != nil {
		ErrorHandler(err, req, res)
	}
}

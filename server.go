package air

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
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
	s.server.ErrorLog = log.New(&serverErrorLogWriter{}, "air: ", 0)

	if DebugMode {
		DEBUG("air: serving in debug mode")
	}

	host := s.server.Addr
	if strings.Contains(host, ":") {
		var err error
		if host, _, err = net.SplitHostPort(host); err != nil {
			return err
		}
	}

	h2hss := &http.Server{
		Addr: host + ":http",
		Handler: http.HandlerFunc(func(
			rw http.ResponseWriter,
			r *http.Request,
		) {
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = r.Host
			}

			http.Redirect(rw, r, "https://"+host+r.RequestURI, 301)
		}),
	}
	defer func() {
		h2hss.Close() // Close anyway, even if it doesn't start
	}()

	if TLSCertFile != "" && TLSKeyFile != "" {
		s.server.TLSConfig = &tls.Config{}
		if HTTPSEnforced {
			go h2hss.ListenAndServe()
		}

		return s.server.ListenAndServeTLS(TLSCertFile, TLSKeyFile)
	} else if DebugMode || !ACMEEnabled {
		return s.server.ListenAndServe()
	}

	acm := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(ACMECertRoot),
		HostPolicy: func(_ context.Context, h string) error {
			if len(HostWhitelist) == 0 ||
				stringsContainsCIly(HostWhitelist, h) {
				return nil
			}

			return fmt.Errorf(
				"acme/autocert: host %q not "+
					"configured in HostWhitelist",
				h,
			)
		},
		Email: MaintainerEmail,
	}

	s.server.Addr = host + ":https"
	s.server.TLSConfig = acm.TLSConfig()

	h2hss.Handler = acm.HTTPHandler(h2hss.Handler)
	go h2hss.ListenAndServe()

	return s.server.ListenAndServeTLS("", "")
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
	// Check host

	if !DebugMode && len(HostWhitelist) > 0 {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}

		// See RFC 3986, section 3.2.2.
		if !stringsContainsCIly(HostWhitelist, host) {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			http.Redirect(
				rw,
				r,
				scheme+"://"+HostWhitelist[0]+r.RequestURI,
				301,
			)

			return
		}
	}

	// Request

	req := &Request{
		Method:        r.Method,
		Scheme:        "http",
		Authority:     r.Host,
		Path:          r.RequestURI,
		Body:          r.Body,
		ContentLength: r.ContentLength,
		Values:        map[string]interface{}{},

		request:                r,
		parseClientAddressOnce: &sync.Once{},
		parseHeadersOnce:       &sync.Once{},
		parseCookiesOnce:       &sync.Once{},
		parseParamsOnce:        &sync.Once{},
	}
	if r.TLS != nil {
		req.Scheme = "https"
	}

	// Response

	res := &Response{
		Status: 200,

		request: req,
		writer:  rw,
	}
	res.Body = &responseBody{
		response: res,
	}

	req.response = res

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

	// Close opened request param file values

	for _, p := range req.params {
		for _, pv := range p.Values {
			if pv.f != nil && pv.f.f != nil {
				pv.f.f.Close()
			}
		}
	}
}

// serverErrorLogWriter is an HTTP server error log writer.
type serverErrorLogWriter struct{}

// Write implements the `io.Writer`.
func (selw *serverErrorLogWriter) Write(b []byte) (int, error) {
	ERROR(string(b))
	return len(b), nil
}

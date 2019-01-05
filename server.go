package air

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// server is an HTTP server.
type server struct {
	a      *Air
	server *http.Server
}

// newServer returns a new instance of the `server` with the a.
func newServer(a *Air) *server {
	return &server{
		a:      a,
		server: &http.Server{},
	}
}

// serve starts the s.
func (s *server) serve() error {
	h2cs := &http2.Server{}
	if s.a.IdleTimeout != 0 {
		h2cs.IdleTimeout = s.a.IdleTimeout
	} else {
		h2cs.IdleTimeout = s.a.ReadTimeout
	}

	s.server.Addr = s.a.Address
	s.server.Handler = h2c.NewHandler(s, h2cs)
	s.server.ReadTimeout = s.a.ReadTimeout
	s.server.ReadHeaderTimeout = s.a.ReadHeaderTimeout
	s.server.WriteTimeout = s.a.WriteTimeout
	s.server.IdleTimeout = s.a.IdleTimeout
	s.server.MaxHeaderBytes = s.a.MaxHeaderBytes
	s.server.ErrorLog = s.a.errorLogger

	if s.a.DebugMode {
		s.a.DEBUG("air: serving in debug mode")
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

			http.Redirect(
				rw,
				r,
				"https://"+host+r.RequestURI,
				http.StatusMovedPermanently,
			)
		}),
	}
	defer h2hss.Close() // Close anyway, even if it doesn't start

	if s.a.TLSCertFile != "" && s.a.TLSKeyFile != "" {
		s.server.TLSConfig = &tls.Config{}
		if s.a.HTTPSEnforced {
			go h2hss.ListenAndServe()
		}

		return s.server.ListenAndServeTLS(
			s.a.TLSCertFile,
			s.a.TLSKeyFile,
		)
	} else if s.a.DebugMode || !s.a.ACMEEnabled {
		return s.server.ListenAndServe()
	}

	acm := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(s.a.ACMECertRoot),
		HostPolicy: func(_ context.Context, h string) error {
			if len(s.a.HostWhitelist) == 0 ||
				stringSliceContainsCIly(s.a.HostWhitelist, h) {
				return nil
			}

			return fmt.Errorf(
				"acme/autocert: host %q not "+
					"configured in HostWhitelist",
				h,
			)
		},
		Email: s.a.MaintainerEmail,
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
	// Check host.

	if !s.a.DebugMode && len(s.a.HostWhitelist) > 0 {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}

		// See RFC 3986, section 3.2.2.
		if !stringSliceContainsCIly(s.a.HostWhitelist, host) {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			http.Redirect(
				rw,
				r,
				scheme+"://"+s.a.HostWhitelist[0]+r.RequestURI,
				http.StatusMovedPermanently,
			)

			return
		}
	}

	// Make request.

	req := &Request{
		Air: s.a,

		parseRouteParamsOnce: &sync.Once{},
		parseOtherParamsOnce: &sync.Once{},
	}
	req.SetHTTPRequest(r)

	// Make response.

	res := &Response{
		Air:    s.a,
		Status: http.StatusOK,

		req:  req,
		ohrw: rw,
	}
	res.SetHTTPResponseWriter(&responseWriter{
		r: res,
		w: rw,
	})

	req.res = res

	// Chain gases.

	h := func(req *Request, res *Response) error {
		rh := s.a.router.route(req)
		h := func(req *Request, res *Response) error {
			if err := rh(req, res); err != nil {
				return err
			} else if !res.Written {
				res.Status = http.StatusNoContent
				r.Header.Del("Content-Type")
				r.Header.Del("Content-Length")
				return res.Write(nil)
			}

			return nil
		}

		for i := len(s.a.Gases) - 1; i >= 0; i-- {
			h = s.a.Gases[i](h)
		}

		return h(req, res)
	}

	// Chain pregases.

	for i := len(s.a.Pregases) - 1; i >= 0; i-- {
		h = s.a.Pregases[i](h)
	}

	// Execute chain.

	if err := h(req, res); err != nil {
		s.a.ErrorHandler(err, req, res)
	}

	// Execute deferred functions.

	if l := len(res.deferredFuncs); l > 0 {
		for i := l - 1; i >= 0; i-- {
			res.deferredFuncs[i]()
		}
	}

	// Put route param values back to the pool.

	if req.routeParamValues != nil {
		s.a.router.routeParamValuesPool.Put(req.routeParamValues)
	}
}

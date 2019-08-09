package air

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// server is an HTTP server.
type server struct {
	a            *Air
	server       *http.Server
	addressMap   map[string]int
	requestPool  *sync.Pool
	responsePool *sync.Pool
}

// newServer returns a new instance of the `server` with the a.
func newServer(a *Air) *server {
	return &server{
		a:          a,
		server:     &http.Server{},
		addressMap: map[string]int{},
		requestPool: &sync.Pool{
			New: func() interface{} {
				return &Request{}
			},
		},
		responsePool: &sync.Pool{
			New: func() interface{} {
				return &Response{}
			},
		},
	}
}

// serve starts the s.
func (s *server) serve() error {
	host, port, err := net.SplitHostPort(s.a.Address)
	if err != nil {
		return err
	}

	s.server.Addr = host + ":" + port
	s.server.Handler = s
	s.server.ReadTimeout = s.a.ReadTimeout
	s.server.ReadHeaderTimeout = s.a.ReadHeaderTimeout
	s.server.WriteTimeout = s.a.WriteTimeout
	s.server.IdleTimeout = s.a.IdleTimeout
	s.server.MaxHeaderBytes = s.a.MaxHeaderBytes
	s.server.ErrorLog = s.a.ErrorLogger

	realPort := port
	hh := http.Handler(http.HandlerFunc(func(
		rw http.ResponseWriter,
		r *http.Request,
	) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}

		if realPort != "443" {
			host = fmt.Sprint(host, ":", realPort)
		}

		http.Redirect(
			rw,
			r,
			"https://"+host+r.RequestURI,
			http.StatusMovedPermanently,
		)
	}))

	if s.a.DebugMode {
		fmt.Println("air: serving in debug mode")
	}

	if s.a.TLSCertFile != "" && s.a.TLSKeyFile != "" {
		c, err := tls.LoadX509KeyPair(s.a.TLSCertFile, s.a.TLSKeyFile)
		if err != nil {
			return err
		}

		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{c},
		}
	} else if !s.a.DebugMode && s.a.ACMEEnabled {
		acm := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(s.a.ACMECertRoot),
			Client: &acme.Client{
				DirectoryURL: s.a.ACMEDirectoryURL,
			},
			Email: s.a.MaintainerEmail,
		}
		if s.a.ACMEHostWhitelist != nil {
			acm.HostPolicy = autocert.HostWhitelist(
				s.a.ACMEHostWhitelist...,
			)
		}

		hh = acm.HTTPHandler(hh)
		s.a.HTTPSEnforced = true

		s.server.TLSConfig = acm.TLSConfig()
	} else {
		h2s := &http2.Server{
			IdleTimeout: s.a.IdleTimeout,
		}
		if h2s.IdleTimeout == 0 {
			h2s.IdleTimeout = s.a.ReadTimeout
		}

		s.server.Handler = h2c.NewHandler(s.server.Handler, h2s)

		l := newListener(s.a)
		if err := l.listen(s.server.Addr); err != nil {
			return err
		}
		defer l.Close()

		s.addressMap[l.Addr().String()] = 0
		defer delete(s.addressMap, l.Addr().String())

		if realPort == "0" {
			_, realPort, _ = net.SplitHostPort(l.Addr().String())
			fmt.Printf("air: listening on %v\n", s.addresses())
		}

		return s.server.Serve(l)
	}

	if s.a.HTTPSEnforced {
		hs := &http.Server{
			Addr:              host + ":" + s.a.HTTPSEnforcedPort,
			Handler:           hh,
			ReadTimeout:       s.a.ReadTimeout,
			ReadHeaderTimeout: s.a.ReadHeaderTimeout,
			WriteTimeout:      s.a.WriteTimeout,
			IdleTimeout:       s.a.IdleTimeout,
			MaxHeaderBytes:    s.a.MaxHeaderBytes,
			ErrorLog:          s.a.ErrorLogger,
		}

		l := newListener(s.a)
		if err := l.listen(hs.Addr); err != nil {
			return err
		}
		defer l.Close()

		s.addressMap[l.Addr().String()] = 1
		defer delete(s.addressMap, l.Addr().String())

		go hs.Serve(l)
		defer hs.Close()
	}

	l := newListener(s.a)
	if err := l.listen(s.server.Addr); err != nil {
		return err
	}
	defer l.Close()

	s.addressMap[l.Addr().String()] = 0
	defer delete(s.addressMap, l.Addr().String())

	if realPort == "0" {
		_, realPort, _ = net.SplitHostPort(l.Addr().String())
		fmt.Printf("air: listening on %v\n", s.addresses())
	}

	return s.server.ServeTLS(l, "", "")
}

// close closes the s immediately.
func (s *server) close() error {
	return s.server.Close()
}

// shutdown gracefully shuts down the s without interrupting any active
// connections.
func (s *server) shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// addresses returns all TCP addresses that the s actually listens on.
func (s *server) addresses() []string {
	asl := len(s.addressMap)
	if asl == 0 {
		return nil
	}

	as := make([]string, 0, asl)
	for a := range s.addressMap {
		as = append(as, a)
	}

	sort.Slice(as, func(i, j int) bool {
		iw := s.addressMap[as[i]]
		jw := s.addressMap[as[j]]
		return iw < jw
	})

	return as
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Get request and response from the pool.

	req := s.requestPool.Get().(*Request)
	res := s.responsePool.Get().(*Response)

	// Tie the request body and the standard request body together.

	r.Body = &requestBody{
		r:  req,
		hr: r,
		rc: r.Body,
	}

	// Reset the request.

	req.Air = s.a
	req.SetHTTPRequest(r)
	req.res = res
	req.params = req.params[:0]
	req.routeParamNames = nil
	req.routeParamValues = nil
	req.parseRouteParamsOnce = &sync.Once{}
	req.parseOtherParamsOnce = &sync.Once{}
	for key := range req.values {
		delete(req.values, key)
	}

	req.localizedString = nil

	// Reset the response.

	res.Air = s.a
	res.SetHTTPResponseWriter(&responseWriter{
		r:  res,
		rw: rw,
	})
	res.Status = http.StatusOK
	res.ContentLength = -1
	res.Written = false
	res.Minified = false
	res.Gzipped = false
	res.req = req
	res.ohrw = rw
	res.servingContent = false
	res.serveContentError = nil
	res.reverseProxying = false
	res.deferredFuncs = res.deferredFuncs[:0]

	// Chain the gases stack.

	h := func(req *Request, res *Response) error {
		h := s.a.router.route(req)
		for i := len(s.a.Gases) - 1; i >= 0; i-- {
			h = s.a.Gases[i](h)
		}

		return h(req, res)
	}

	// Chain the pregases stack.

	for i := len(s.a.Pregases) - 1; i >= 0; i-- {
		h = s.a.Pregases[i](h)
	}

	// Execute the chain.

	if err := h(req, res); err != nil {
		if !res.Written && res.Status < http.StatusBadRequest {
			res.Status = http.StatusInternalServerError
		}

		s.a.ErrorHandler(err, req, res)
	}

	// Execute the deferred functions.

	for i := len(res.deferredFuncs) - 1; i >= 0; i-- {
		res.deferredFuncs[i]()
	}

	// Put the route param values back to the pool.

	if req.routeParamValues != nil {
		s.a.router.routeParamValuesPool.Put(req.routeParamValues)
	}

	// Put the request and response back to the pool.

	s.requestPool.Put(req)
	s.responsePool.Put(res)
}

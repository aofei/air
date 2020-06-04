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
	a                *Air
	server           *http.Server
	addressMap       map[string]int
	shutdownJobs     []func()
	shutdownJobMutex *sync.Mutex
	shutdownJobDone  chan struct{}
	requestPool      *sync.Pool
	responsePool     *sync.Pool
}

// newServer returns a new instance of the `server` with the a.
func newServer(a *Air) *server {
	return &server{
		a:                a,
		server:           &http.Server{},
		addressMap:       map[string]int{},
		shutdownJobMutex: &sync.Mutex{},
		shutdownJobDone:  make(chan struct{}),
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

	s.server.Addr = net.JoinHostPort(host, port)
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
			host = net.JoinHostPort(host, realPort)
		}

		http.Redirect(
			rw,
			r,
			"https://"+host+r.RequestURI,
			http.StatusMovedPermanently,
		)
	}))

	shutdownJobRunOnce := sync.Once{}
	s.server.RegisterOnShutdown(func() {
		s.shutdownJobMutex.Lock()
		defer s.shutdownJobMutex.Unlock()
		shutdownJobRunOnce.Do(func() {
			waitGroup := sync.WaitGroup{}
			for _, job := range s.shutdownJobs {
				if job != nil {
					waitGroup.Add(1)
					go func(job func()) {
						job()
						waitGroup.Done()
					}(job)
				}
			}

			waitGroup.Wait()

			close(s.shutdownJobDone)
		})
	})

	if s.a.DebugMode {
		fmt.Println("air: serving in debug mode")
	}

	tlsConfig := s.a.TLSConfig
	if tlsConfig != nil {
		tlsConfig = tlsConfig.Clone()
	}

	if s.a.TLSCertFile != "" && s.a.TLSKeyFile != "" {
		c, err := tls.LoadX509KeyPair(s.a.TLSCertFile, s.a.TLSKeyFile)
		if err != nil {
			return err
		}

		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, c)
	}

	if tlsConfig != nil {
		for _, proto := range []string{"h2", "http/1.1"} {
			if !stringSliceContains(
				tlsConfig.NextProtos,
				proto,
				false,
			) {
				tlsConfig.NextProtos = append(
					tlsConfig.NextProtos,
					proto,
				)
			}
		}
	}

	if s.a.ACMEEnabled {
		acm := &autocert.Manager{
			Prompt: func(tosURL string) bool {
				if len(s.a.ACMETOSURLWhitelist) == 0 {
					return true
				}

				for _, u := range s.a.ACMETOSURLWhitelist {
					if u == tosURL {
						return true
					}
				}

				return false
			},
			Cache:       autocert.DirCache(s.a.ACMECertRoot),
			RenewBefore: s.a.ACMERenewalWindow,
			Client: &acme.Client{
				Key:          s.a.ACMEAccountKey,
				DirectoryURL: s.a.ACMEDirectoryURL,
			},
			Email:           s.a.MaintainerEmail,
			ExtraExtensions: s.a.ACMEExtraExts,
		}
		if s.a.ACMEHostWhitelist != nil {
			acm.HostPolicy = autocert.HostWhitelist(
				s.a.ACMEHostWhitelist...,
			)
		}

		hh = acm.HTTPHandler(hh)

		acmTLSConfig := acm.TLSConfig()

		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}

		getCertificate := tlsConfig.GetCertificate
		tlsConfig.GetCertificate = func(
			chi *tls.ClientHelloInfo,
		) (*tls.Certificate, error) {
			if getCertificate != nil {
				c, err := getCertificate(chi)
				if err != nil {
					return nil, err
				}

				if c != nil {
					return c, nil
				}
			}

			if chi.ServerName == "" {
				chi.ServerName = s.a.ACMEDefaultHost
			}

			return acm.GetCertificate(chi)
		}

		for _, proto := range acmTLSConfig.NextProtos {
			if !stringSliceContains(
				tlsConfig.NextProtos,
				proto,
				false,
			) {
				tlsConfig.NextProtos = append(
					tlsConfig.NextProtos,
					proto,
				)
			}
		}
	}

	listener := newListener(s.a)
	if err := listener.listen(s.server.Addr); err != nil {
		return err
	}
	defer listener.Close()

	s.addressMap[listener.Addr().String()] = 0
	defer delete(s.addressMap, listener.Addr().String())

	netListener := net.Listener(listener)
	httpsEnforced := s.a.ACMEEnabled || s.a.HTTPSEnforced
	if tlsConfig != nil {
		netListener = tls.NewListener(netListener, tlsConfig)
		if httpsEnforced {
			hs := &http.Server{
				Addr: net.JoinHostPort(
					host,
					s.a.HTTPSEnforcedPort,
				),
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
	} else {
		h2s := &http2.Server{
			IdleTimeout: s.a.IdleTimeout,
		}
		if h2s.IdleTimeout == 0 {
			h2s.IdleTimeout = s.a.ReadTimeout
		}

		s.server.Handler = h2c.NewHandler(s.server.Handler, h2s)
	}

	if realPort == "0" || (httpsEnforced && s.a.HTTPSEnforcedPort == "0") {
		_, realPort, _ = net.SplitHostPort(netListener.Addr().String())
		fmt.Printf("air: listening on %v\n", s.addresses())
	}

	return s.server.Serve(netListener)
}

// close closes the s immediately.
func (s *server) close() error {
	return s.server.Close()
}

// shutdown gracefully shuts down the s without interrupting any active
// connections.
func (s *server) shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.shutdownJobDone:
	}

	return err
}

// addShutdownJob adds the f to the shutdown job queue and returns an unique ID.
func (s *server) addShutdownJob(f func()) int {
	s.shutdownJobMutex.Lock()
	defer s.shutdownJobMutex.Unlock()
	s.shutdownJobs = append(s.shutdownJobs, f)
	return len(s.shutdownJobs) - 1
}

// removeShutdownJob removes the shutdown job targeted by the id from the
// shutdown job queue.
func (s *server) removeShutdownJob(id int) {
	s.shutdownJobMutex.Lock()
	defer s.shutdownJobMutex.Unlock()
	if id >= 0 && id < len(s.shutdownJobs) {
		s.shutdownJobs[id] = nil
	}
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
	// Get request from the pool.

	req := s.requestPool.Get().(*Request)
	req.Air = s.a
	req.params = req.params[:0]
	req.routeParamNames = nil
	req.routeParamValues = nil
	req.parseRouteParamsOnce = &sync.Once{}
	req.parseOtherParamsOnce = &sync.Once{}
	for key := range req.values {
		delete(req.values, key)
	}

	req.localizedString = nil

	req.SetHTTPRequest(r)

	// Get response from the pool.

	res := s.responsePool.Get().(*Response)
	res.Air = s.a
	res.Status = http.StatusOK
	res.ContentLength = -1
	res.Written = false
	res.Minified = false
	res.Gzipped = false
	res.servingContent = false
	res.serveContentError = nil
	res.reverseProxying = false
	res.deferredFuncs = res.deferredFuncs[:0]

	hrw := http.ResponseWriter(&responseWriter{
		r:  res,
		rw: rw,
	})

	if hijacker, ok := rw.(http.Hijacker); ok {
		hrw = http.ResponseWriter(&struct {
			http.ResponseWriter
			http.Hijacker
		}{
			hrw,
			&responseHijacker{
				r: res,
				h: hijacker,
			},
		})
	}

	if pusher, ok := rw.(http.Pusher); ok {
		hrw = http.ResponseWriter(&struct {
			http.ResponseWriter
			http.Pusher
		}{
			hrw,
			pusher,
		})
	}

	res.SetHTTPResponseWriter(hrw)

	// Tie the request and response together.

	req.res = res
	res.req = req

	// Tie the request body and standard request body together.

	r.Body = &requestBody{
		r:  req,
		hr: r,
		rc: r.Body,
	}

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

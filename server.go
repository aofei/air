package air

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// server represents the HTTP server.
//
// It's embedded with `http.Server`.
type server struct {
	*http.Server

	air *Air
}

// newServer returns a pointer of a new instance of `server`.
func newServer(a *Air) *server {
	s := &server{
		Server: &http.Server{},
		air:    a,
	}
	s.Addr = s.air.Config.Address
	s.Handler = s
	s.ReadTimeout = s.air.Config.ReadTimeout
	s.WriteTimeout = s.air.Config.WriteTimeout
	return s
}

// start starts the HTTP server.
func (s *server) start() error {
	cl := s.air.Config.Listener
	if cl != nil {
		return s.Serve(cl)
	}

	ln, err := net.Listen("tcp", s.air.Config.Address)
	if err != nil {
		return err
	}

	ctcf := s.air.Config.TLSCertFile
	ctkf := s.air.Config.TLSKeyFile
	if ctcf != "" && ctkf != "" {
		tlscfg := &tls.Config{}
		if !s.air.Config.DisableHTTP2 {
			tlscfg.NextProtos = append(tlscfg.NextProtos, "h2")
		}
		tlscfg.Certificates = make([]tls.Certificate, 1)
		tlscfg.Certificates[0], err = tls.LoadX509KeyPair(ctcf, ctkf)
		if err != nil {
			return err
		}
		cl = tls.NewListener(
			tcpKeepAliveListener{ln.(*net.TCPListener)},
			tlscfg,
		)
	} else {
		cl = tcpKeepAliveListener{ln.(*net.TCPListener)}
	}

	return s.Serve(cl)
}

// stop stops the HTTP server.
func (s *server) stop() error {
	return s.air.Config.Listener.Close()
}

// ServeHTTP implements the `http.Handler`.
func (s *server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := contextPool.Get().(*Context)
	c.feed(rw, req)

	// Gases
	h := func(c *Context) error {
		method := c.Request.Method
		path := c.Request.URL.EscapedPath()
		s.air.router.route(method, path, c)
		h := c.Handler
		for i := len(s.air.gases) - 1; i >= 0; i-- {
			h = s.air.gases[i](h)
		}
		return h(c)
	}

	// Pregases
	for i := len(s.air.pregases) - 1; i >= 0; i-- {
		h = s.air.pregases[i](h)
	}

	// Execute chain
	if err := h(c); err != nil {
		s.air.HTTPErrorHandler(err, c)
	}

	c.reset()
	contextPool.Put(c)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted connections. It's used by
// ListenAndServe and ListenAndServeTLS so dead TCP connections (e.g. closing laptop mid-download)
// eventually go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept implements the `net.TCPListener#Accept()`.
func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

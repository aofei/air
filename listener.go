package air

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// listener implements the `net.Listener`. It supports the TCP keep-alive and
// the PROXY protocol.
type listener struct {
	*net.TCPListener

	a                                 *Air
	allowedPROXYProtocolRelayerIPNets []*net.IPNet
}

// newListener returns a new instance of the `listener` with the a.
func newListener(a *Air) *listener {
	var ipNets []*net.IPNet
	if len(a.PROXYProtocolRelayerIPWhitelist) > 0 {
		for _, s := range a.PROXYProtocolRelayerIPWhitelist {
			if ip := net.ParseIP(s); ip != nil {
				s = ip.String()
				switch {
				case ip.IsUnspecified():
					s += "/0"
				case ip.To4() != nil: // IPv4
					s += "/32"
				case ip.To16() != nil: // IPv6
					s += "/128"
				}
			}

			if _, ipNet, _ := net.ParseCIDR(s); ipNet != nil {
				ipNets = append(ipNets, ipNet)
			}
		}
	}

	return &listener{
		a:                                 a,
		allowedPROXYProtocolRelayerIPNets: ipNets,
	}
}

// listen listens on the TCP network address.
func (l *listener) listen(address string) error {
	nl, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	l.TCPListener = nl.(*net.TCPListener)

	return nil
}

// Accept implements the `net.Listener`.
func (l *listener) Accept() (net.Conn, error) {
	tc, err := l.AcceptTCP()
	if err != nil {
		return nil, err
	}

	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)

	if !l.a.PROXYProtocolEnabled {
		return tc, nil
	}

	proxyable := len(l.allowedPROXYProtocolRelayerIPNets) == 0
	if !proxyable {
		host, _, _ := net.SplitHostPort(tc.RemoteAddr().String())
		ip := net.ParseIP(host)
		for _, ipNet := range l.allowedPROXYProtocolRelayerIPNets {
			if ipNet.Contains(ip) {
				proxyable = true
				break
			}
		}
	}

	if proxyable {
		return &proxyProtocolConn{
			Conn:              tc,
			bufReader:         bufio.NewReader(tc),
			readHeaderOnce:    &sync.Once{},
			readHeaderTimeout: l.a.PROXYProtocolReadHeaderTimeout,
		}, nil
	}

	return tc, nil
}

// proxyProtocolConn implements the `net.Conn`. It is used to wrap a `net.Conn`
// which may be speaking the PROXY protocol.
type proxyProtocolConn struct {
	net.Conn

	bufReader         *bufio.Reader
	srcAddr           *net.TCPAddr
	dstAddr           *net.TCPAddr
	readHeaderOnce    *sync.Once
	readHeaderError   error
	readHeaderTimeout time.Duration
}

// Read implements the `net.Conn`.
func (ppc *proxyProtocolConn) Read(b []byte) (int, error) {
	ppc.readHeaderOnce.Do(ppc.readHeader)
	if ppc.readHeaderError != nil {
		return 0, ppc.readHeaderError
	}

	return ppc.bufReader.Read(b)
}

// LocalAddr implements the `net.Conn`.
func (ppc *proxyProtocolConn) LocalAddr() net.Addr {
	ppc.readHeaderOnce.Do(ppc.readHeader)
	if ppc.dstAddr != nil {
		return ppc.dstAddr
	}

	return ppc.Conn.LocalAddr()
}

// RemoteAddr implements the `net.Conn`.
func (ppc *proxyProtocolConn) RemoteAddr() net.Addr {
	ppc.readHeaderOnce.Do(ppc.readHeader)
	if ppc.srcAddr != nil {
		return ppc.srcAddr
	}

	return ppc.Conn.RemoteAddr()
}

// readHeader reads the PROXY protocol header. It does nothing if the
// underlying connection is not speaking the PROXY protocol.
func (ppc *proxyProtocolConn) readHeader() {
	if ppc.readHeaderTimeout != 0 {
		ppc.SetReadDeadline(time.Now().Add(ppc.readHeaderTimeout))
		defer ppc.SetReadDeadline(time.Time{})
	}

	defer func() {
		if ppc.readHeaderError != nil && ppc.readHeaderError != io.EOF {
			ppc.Close()
			ppc.bufReader = bufio.NewReader(ppc.Conn)
		}
	}()

	for i := 0; i < 6; i++ { // i < len("PROXY ")
		var b []byte
		b, ppc.readHeaderError = ppc.bufReader.Peek(i + 1)
		if ppc.readHeaderError != nil {
			ne, ok := ppc.readHeaderError.(net.Error)
			if ok && ne.Timeout() {
				ppc.readHeaderError = nil
				return
			}

			return
		}

		if b[i] != "PROXY "[i] { // Not speaking the PROXY protocol
			return
		}
	}

	var header string
	header, ppc.readHeaderError = ppc.bufReader.ReadString('\n')
	if ppc.readHeaderError != nil {
		return
	}

	header = header[:len(header)-2] // Strip CRLF

	// PROXY <protocol> <src ip> <dst ip> <src port> <dst port>
	parts := strings.Split(header, " ")
	if len(parts) != 6 {
		ppc.readHeaderError = fmt.Errorf(
			"air: malformed proxy header line: %s",
			header,
		)
		return
	}

	switch parts[1] { // <protocol>
	case "TCP4", "TCP6":
	default:
		ppc.readHeaderError = fmt.Errorf(
			"air: unsupported proxy protocol: %s",
			parts[1],
		)
		return
	}

	srcIP := net.ParseIP(parts[2]) // <src ip>
	if srcIP == nil {
		ppc.readHeaderError = fmt.Errorf(
			"air: invalid proxy source ip: %s",
			parts[2],
		)
		return
	}

	dstIP := net.ParseIP(parts[3]) // <dst ip>
	if dstIP == nil {
		ppc.readHeaderError = fmt.Errorf(
			"air: invalid proxy destination ip: %s",
			parts[3],
		)
		return
	}

	srcPort, err := strconv.Atoi(parts[4]) // <src port>
	if err != nil {
		ppc.readHeaderError = fmt.Errorf(
			"air: invalid proxy source port: %s",
			parts[4],
		)
		return
	}

	dstPort, err := strconv.Atoi(parts[5]) // <dst port>
	if err != nil {
		ppc.readHeaderError = fmt.Errorf(
			"air: invalid proxy destination port: %s",
			parts[5],
		)
		return
	}

	ppc.srcAddr = &net.TCPAddr{IP: srcIP, Port: srcPort}
	ppc.dstAddr = &net.TCPAddr{IP: dstIP, Port: dstPort}

	return
}

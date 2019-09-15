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

	a                         *Air
	allowedPROXYRelayerIPNets []*net.IPNet
}

// newListener returns a new instance of the `listener` with the a.
func newListener(a *Air) *listener {
	var ipNets []*net.IPNet
	if len(a.PROXYRelayerIPWhitelist) > 0 {
		for _, s := range a.PROXYRelayerIPWhitelist {
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
		a:                         a,
		allowedPROXYRelayerIPNets: ipNets,
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

	if !l.a.PROXYEnabled {
		return tc, nil
	}

	proxyable := len(l.allowedPROXYRelayerIPNets) == 0
	if !proxyable {
		host, _, _ := net.SplitHostPort(tc.RemoteAddr().String())
		ip := net.ParseIP(host)
		for _, ipNet := range l.allowedPROXYRelayerIPNets {
			if ipNet.Contains(ip) {
				proxyable = true
				break
			}
		}
	}

	if proxyable {
		return &proxyConn{
			Conn:              tc,
			bufReader:         bufio.NewReader(tc),
			readHeaderOnce:    &sync.Once{},
			readHeaderTimeout: l.a.PROXYReadHeaderTimeout,
		}, nil
	}

	return tc, nil
}

// proxyConn implements the `net.Conn`. It is used to wrap a `net.Conn` which
// may be speaking the PROXY protocol.
type proxyConn struct {
	net.Conn

	bufReader         *bufio.Reader
	srcAddr           *net.TCPAddr
	dstAddr           *net.TCPAddr
	readHeaderOnce    *sync.Once
	readHeaderError   error
	readHeaderTimeout time.Duration
}

// Read implements the `net.Conn`.
func (pc *proxyConn) Read(b []byte) (int, error) {
	pc.readHeaderOnce.Do(pc.readHeader)
	if pc.readHeaderError != nil {
		return 0, pc.readHeaderError
	}

	return pc.bufReader.Read(b)
}

// LocalAddr implements the `net.Conn`.
func (pc *proxyConn) LocalAddr() net.Addr {
	pc.readHeaderOnce.Do(pc.readHeader)
	if pc.dstAddr != nil {
		return pc.dstAddr
	}

	return pc.Conn.LocalAddr()
}

// RemoteAddr implements the `net.Conn`.
func (pc *proxyConn) RemoteAddr() net.Addr {
	pc.readHeaderOnce.Do(pc.readHeader)
	if pc.srcAddr != nil {
		return pc.srcAddr
	}

	return pc.Conn.RemoteAddr()
}

// readHeader reads the PROXY protocol header. It does nothing if the underlying
// connection is not speaking the PROXY protocol.
func (pc *proxyConn) readHeader() {
	if pc.readHeaderTimeout != 0 {
		pc.SetReadDeadline(time.Now().Add(pc.readHeaderTimeout))
		defer pc.SetReadDeadline(time.Time{})
	}

	defer func() {
		if pc.readHeaderError != nil && pc.readHeaderError != io.EOF {
			pc.Close()
			pc.bufReader = bufio.NewReader(pc.Conn)
		}
	}()

	for i := 0; i < 6; i++ { // i < len("PROXY ")
		var b []byte
		b, pc.readHeaderError = pc.bufReader.Peek(i + 1)
		if pc.readHeaderError != nil {
			ne, ok := pc.readHeaderError.(net.Error)
			if ok && ne.Timeout() {
				pc.readHeaderError = nil
				return
			}

			return
		}

		if b[i] != "PROXY "[i] { // Not speaking the PROXY protocol
			return
		}
	}

	var header string
	header, pc.readHeaderError = pc.bufReader.ReadString('\n')
	if pc.readHeaderError != nil {
		return
	}

	header = header[:len(header)-2] // Strip CRLF

	// PROXY <protocol> <src ip> <dst ip> <src port> <dst port>
	parts := strings.Split(header, " ")
	if len(parts) != 6 {
		pc.readHeaderError = fmt.Errorf(
			"air: malformed proxy header line: %s",
			header,
		)
		return
	}

	switch parts[1] { // <protocol>
	case "TCP4", "TCP6":
	default:
		pc.readHeaderError = fmt.Errorf(
			"air: unsupported proxy protocol: %s",
			parts[1],
		)
		return
	}

	srcIP := net.ParseIP(parts[2]) // <src ip>
	if srcIP == nil {
		pc.readHeaderError = fmt.Errorf(
			"air: invalid proxy source ip: %s",
			parts[2],
		)
		return
	}

	dstIP := net.ParseIP(parts[3]) // <dst ip>
	if dstIP == nil {
		pc.readHeaderError = fmt.Errorf(
			"air: invalid proxy destination ip: %s",
			parts[3],
		)
		return
	}

	srcPort, err := strconv.Atoi(parts[4]) // <src port>
	if err != nil {
		pc.readHeaderError = fmt.Errorf(
			"air: invalid proxy source port: %s",
			parts[4],
		)
		return
	}

	dstPort, err := strconv.Atoi(parts[5]) // <dst port>
	if err != nil {
		pc.readHeaderError = fmt.Errorf(
			"air: invalid proxy destination port: %s",
			parts[5],
		)
		return
	}

	pc.srcAddr = &net.TCPAddr{IP: srcIP, Port: srcPort}
	pc.dstAddr = &net.TCPAddr{IP: dstIP, Port: dstPort}

	return
}

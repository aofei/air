package air

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// proxyProtocolSign is the signature of the PROXY protocol.
var proxyProtocolSign = []byte{
	0x0d, 0x0a, 0x0d, 0x0a,
	0x00, 0x0d, 0x0a, 0x51,
	0x55, 0x49, 0x54, 0x0a,
}

// listener implements the `net.Listener`. It supports the TCP keep-alive and
// PROXY protocol.
type listener struct {
	*net.TCPListener

	a                         *Air
	allowedPROXYRelayerIPNets []*net.IPNet
}

// newListener returns a new instance of the `listener` with the a.
func newListener(a *Air) *listener {
	var ipNets []*net.IPNet
	for _, s := range a.PROXYRelayerIPWhitelist {
		if ip := net.ParseIP(s); ip != nil {
			s = ip.String()
			switch {
			case ip.IsUnspecified():
				s += "/0"
			case ip.To4() != nil:
				s += "/32"
			case ip.To16() != nil:
				s += "/128"
			}
		}

		if _, ipNet, _ := net.ParseCIDR(s); ipNet != nil {
			ipNets = append(ipNets, ipNet)
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

// readHeader reads the PROXY protocol header. It does nothing if the connection
// of the pc is not speaking the PROXY protocol.
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

	isV1 := true
	for i := 0; i < 6; i++ { // i < len("PROXY ")
		var b []byte
		b, pc.readHeaderError = pc.bufReader.Peek(i + 1)
		if pc.readHeaderError != nil {
			var ne net.Error
			if errors.As(pc.readHeaderError, &ne) && ne.Timeout() {
				pc.readHeaderError = nil
				return
			}

			return
		}

		// Check if it is speaking the PROXY protocol version 1.
		if b[i] != "PROXY "[i] {
			isV1 = false
			break
		}
	}

	if isV1 {
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
				"air: unsupported proxy transport protocol: %s",
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

		pc.srcAddr = &net.TCPAddr{
			IP:   srcIP,
			Port: srcPort,
		}

		pc.dstAddr = &net.TCPAddr{
			IP:   dstIP,
			Port: dstPort,
		}

		return
	}

	for i := 0; i < len(proxyProtocolSign); i++ {
		var b []byte
		b, pc.readHeaderError = pc.bufReader.Peek(i + 1)
		if pc.readHeaderError != nil {
			var ne net.Error
			if errors.As(pc.readHeaderError, &ne) && ne.Timeout() {
				pc.readHeaderError = nil
				return
			}

			return
		}

		// Check if it is speaking the PROXY protocol.
		if b[i] != proxyProtocolSign[i] {
			return
		}
	}

	_, pc.readHeaderError = pc.bufReader.Discard(len(proxyProtocolSign))
	if pc.readHeaderError != nil {
		return
	}

	// Protocol version and command.

	var b byte
	b, pc.readHeaderError = pc.bufReader.ReadByte()
	if b&0xf0 != 0x20 { // 2
		pc.readHeaderError = errors.New(
			"air: unsupported proxy protocol version",
		)
		return
	} else if b&0x0f != 0x01 { // PROXY
		pc.readHeaderError = errors.New(
			"air: unsupported proxy command",
		)
		return
	}

	// Address family and transport protocol.

	b, pc.readHeaderError = pc.bufReader.ReadByte()
	switch b & 0xf0 {
	case 0x10: // AF_INET
	case 0x20: // AF_INET6
	default:
		pc.readHeaderError = errors.New(
			"air: unsupported proxy address family",
		)
		return
	}

	if b&0x0f != 0x01 { // STREAM
		pc.readHeaderError = errors.New(
			"air: unsupported proxy transport protocol",
		)
		return
	}

	var expectedAddressLength uint16
	switch b {
	case 0x11: // TCP over IPv4
		expectedAddressLength = 12
	case 0x21: // TCP over IPv6
		expectedAddressLength = 36
	default:
		pc.readHeaderError = errors.New(
			"air: unsupported combination of proxy address " +
				"family and transport protocol",
		)
		return
	}

	// Address length.

	var addressLength uint16
	if err := binary.Read(
		io.LimitReader(pc.bufReader, 2),
		binary.BigEndian,
		&addressLength,
	); err != nil {
		pc.readHeaderError = fmt.Errorf(
			"air: failed to read proxy address length: %v",
			err,
		)
		return
	}

	if addressLength != expectedAddressLength {
		pc.readHeaderError = fmt.Errorf(
			"air: invalid proxy address length: %d",
			addressLength,
		)
		return
	}

	if _, err := pc.bufReader.Peek(int(addressLength)); err != nil {
		pc.readHeaderError = fmt.Errorf(
			"air: failed to peek proxy addresses and ports: %v",
			err,
		)
		return
	}

	var srcIP, dstIP net.IP
	switch addressLength {
	case 12: // TCP over IPv4
		srcIP, dstIP = make(net.IP, 4), make(net.IP, 4)
	case 36: // TCP over IPv6
		srcIP, dstIP = make(net.IP, 16), make(net.IP, 16)
	}

	var srcPort, dstPort = make([]byte, 2), make([]byte, 2)

	if err := binary.Read(
		io.LimitReader(pc.bufReader, int64(addressLength)),
		binary.BigEndian,
		append(srcIP, append(dstIP, append(srcPort, dstPort...)...)...),
	); err != nil {
		pc.readHeaderError = fmt.Errorf(
			"air: failed to read proxy addresses and ports: %v",
			err,
		)
		return
	}

	pc.srcAddr = &net.TCPAddr{
		IP:   srcIP,
		Port: int(binary.BigEndian.Uint16(srcPort)),
	}

	pc.dstAddr = &net.TCPAddr{
		IP:   dstIP,
		Port: int(binary.BigEndian.Uint16(dstPort)),
	}
}

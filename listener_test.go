package air

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewListener(t *testing.T) {
	a := New()
	a.PROXYProtocolEnabled = true

	l := newListener(a)

	assert.NotNil(t, l)
	assert.Nil(t, l.TCPListener)
	assert.NotNil(t, l.a)
	assert.Nil(t, l.allowedPROXYProtocolRelayerIPNets)

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolRelayerIPWhitelist = []string{
		"0.0.0.0",
		"::",
		"127.0.0.1",
		"127.0.0.1/32",
		"::1",
		"::1/128",
	}

	l = newListener(a)

	assert.NotNil(t, l)
	assert.Nil(t, l.TCPListener)
	assert.NotNil(t, l.a)
	assert.Len(t, l.allowedPROXYProtocolRelayerIPNets, 6)
}

func TestListenerListen(t *testing.T) {
	a := New()
	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))
	assert.NoError(t, l.Close())

	a = New()
	l = newListener(a)

	assert.Error(t, l.listen(":-1"))
}

func TestListenerAccept(t *testing.T) {
	a := New()
	l := newListener(a)

	c, err := l.Accept()
	assert.Nil(t, c)
	assert.Error(t, err)

	a = New()
	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok := c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)
	assert.NotNil(t, ppc.Conn)
	assert.NotNil(t, ppc.bufReader)
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.NotNil(t, ppc.readHeaderOnce)
	assert.Nil(t, ppc.readHeaderError)
	assert.Zero(t, ppc.readHeaderTimeout)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolRelayerIPWhitelist = []string{"127.0.0.1"}

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolRelayerIPWhitelist = []string{"127.0.0.2"}

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	assert.NoError(t, l.Close())
}

func TestPROXYProtocolConnRead(t *testing.T) {
	a := New()
	a.PROXYProtocolEnabled = true

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok := c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	b := make([]byte, 3)
	n, err := ppc.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "air", string(b))

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY "))
		cc.Close()
	}()

	b = make([]byte, 6)
	n, err = ppc.Read(b)
	assert.Zero(t, n)
	assert.Error(t, err)
	assert.Equal(t, "\x00\x00\x00\x00\x00\x00", string(b))

	assert.NoError(t, l.Close())
}

func TestPROXYProtocolConnLocalAddr(t *testing.T) {
	a := New()
	a.PROXYProtocolEnabled = true

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok := c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	b := make([]byte, 3)
	n, err := ppc.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "air", string(b))

	na := ppc.LocalAddr()
	assert.NotNil(t, na)
	assert.Equal(t, c.LocalAddr().Network(), na.Network())
	assert.Equal(t, c.LocalAddr().String(), na.String())

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	na = ppc.LocalAddr()
	assert.NotNil(t, na)
	assert.Equal(t, "tcp", na.Network())
	assert.Equal(t, "127.0.0.3:8082", na.String())

	assert.NoError(t, l.Close())
}

func TestPROXYProtocolConnRemoteAddr(t *testing.T) {
	a := New()
	a.PROXYProtocolEnabled = true

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok := c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	b := make([]byte, 3)
	n, err := ppc.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "air", string(b))

	na := ppc.RemoteAddr()
	assert.NotNil(t, na)
	assert.Equal(t, c.RemoteAddr().Network(), na.Network())
	assert.Equal(t, c.RemoteAddr().String(), na.String())

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	na = ppc.RemoteAddr()
	assert.NotNil(t, na)
	assert.Equal(t, "tcp", na.Network())
	assert.Equal(t, "127.0.0.2:8081", na.String())

	assert.NoError(t, l.Close())
}

func TestPROXYProtocolConnReadHeader(t *testing.T) {
	a := New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok := c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Nil(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.NotNil(t, ppc.srcAddr)
	assert.NotNil(t, ppc.dstAddr)
	assert.NoError(t, ppc.readHeaderError)
	assert.Equal(t, "tcp", ppc.srcAddr.Network())
	assert.Equal(t, "127.0.0.2:8081", ppc.srcAddr.String())
	assert.Equal(t, "tcp", ppc.dstAddr.Network())
	assert.Equal(t, "127.0.0.3:8082", ppc.dstAddr.String())

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(200*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		time.Sleep(150 * time.Millisecond)
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.NoError(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	assert.NoError(t, ppc.Close())

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY "))
		time.Sleep(150 * time.Millisecond)
		cc.Write([]byte("TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY UDP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0 8081 8082\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 PORT 8082\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYProtocolEnabled = true
	a.PROXYProtocolReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	ppc, ok = c.(*proxyProtocolConn)
	assert.NotNil(t, ppc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 PORT\r\n"))
		cc.Close()
	}()

	ppc.readHeader()
	assert.Nil(t, ppc.srcAddr)
	assert.Nil(t, ppc.dstAddr)
	assert.Error(t, ppc.readHeaderError)

	assert.NoError(t, l.Close())
}

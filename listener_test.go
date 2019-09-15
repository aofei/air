package air

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewListener(t *testing.T) {
	a := New()
	a.PROXYEnabled = true

	l := newListener(a)

	assert.NotNil(t, l)
	assert.Nil(t, l.TCPListener)
	assert.NotNil(t, l.a)
	assert.Nil(t, l.allowedPROXYRelayerIPNets)

	a = New()
	a.PROXYEnabled = true
	a.PROXYRelayerIPWhitelist = []string{
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
	assert.Len(t, l.allowedPROXYRelayerIPNets, 6)
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
	a.PROXYEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok := c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)
	assert.NotNil(t, pc.Conn)
	assert.NotNil(t, pc.bufReader)
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.NotNil(t, pc.readHeaderOnce)
	assert.Nil(t, pc.readHeaderError)
	assert.Zero(t, pc.readHeaderTimeout)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYRelayerIPWhitelist = []string{"127.0.0.1"}

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
	a.PROXYEnabled = true
	a.PROXYRelayerIPWhitelist = []string{"127.0.0.2"}

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

func TestPROXYConnRead(t *testing.T) {
	a := New()
	a.PROXYEnabled = true

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok := c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	b := make([]byte, 3)
	n, err := pc.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "air", string(b))

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY "))
		cc.Close()
	}()

	b = make([]byte, 6)
	n, err = pc.Read(b)
	assert.Zero(t, n)
	assert.Error(t, err)
	assert.Equal(t, "\x00\x00\x00\x00\x00\x00", string(b))

	assert.NoError(t, l.Close())
}

func TestPROXYConnLocalAddr(t *testing.T) {
	a := New()
	a.PROXYEnabled = true

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok := c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	b := make([]byte, 3)
	n, err := pc.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "air", string(b))

	na := pc.LocalAddr()
	assert.NotNil(t, na)
	assert.Equal(t, c.LocalAddr().Network(), na.Network())
	assert.Equal(t, c.LocalAddr().String(), na.String())

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	na = pc.LocalAddr()
	assert.NotNil(t, na)
	assert.Equal(t, "tcp", na.Network())
	assert.Equal(t, "127.0.0.3:8082", na.String())

	assert.NoError(t, l.Close())
}

func TestPROXYConnRemoteAddr(t *testing.T) {
	a := New()
	a.PROXYEnabled = true

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok := c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	b := make([]byte, 3)
	n, err := pc.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "air", string(b))

	na := pc.RemoteAddr()
	assert.NotNil(t, na)
	assert.Equal(t, c.RemoteAddr().Network(), na.Network())
	assert.Equal(t, c.RemoteAddr().String(), na.String())

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	na = pc.RemoteAddr()
	assert.NotNil(t, na)
	assert.Equal(t, "tcp", na.Network())
	assert.Equal(t, "127.0.0.2:8081", na.String())

	assert.NoError(t, l.Close())
}

func TestPROXYConnReadHeader(t *testing.T) {
	a := New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l := newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err := net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err := l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok := c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("air"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Nil(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.NotNil(t, pc.srcAddr)
	assert.NotNil(t, pc.dstAddr)
	assert.NoError(t, pc.readHeaderError)
	assert.Equal(t, "tcp", pc.srcAddr.Network())
	assert.Equal(t, "127.0.0.2:8081", pc.srcAddr.String())
	assert.Equal(t, "tcp", pc.dstAddr.Network())
	assert.Equal(t, "127.0.0.3:8082", pc.dstAddr.String())

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(200*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		time.Sleep(150 * time.Millisecond)
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.NoError(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	assert.NoError(t, pc.Close())

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY "))
		time.Sleep(150 * time.Millisecond)
		cc.Write([]byte("TCP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY UDP4 127.0.0.2 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0 127.0.0.3 8081 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0 8081 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 PORT 8082\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())

	a = New()
	a.PROXYEnabled = true
	a.PROXYReadHeaderTimeout = 100 * time.Millisecond

	l = newListener(a)

	assert.NoError(t, l.listen("localhost:0"))

	cc, err = net.Dial("tcp", l.Addr().String())
	assert.NotNil(t, cc)
	assert.NoError(t, err)
	assert.NoError(t, cc.SetDeadline(time.Now().Add(100*time.Millisecond)))

	c, err = l.Accept()
	assert.NotNil(t, c)
	assert.NoError(t, err)

	pc, ok = c.(*proxyConn)
	assert.NotNil(t, pc)
	assert.True(t, ok)

	go func() {
		cc.Write([]byte("PROXY TCP4 127.0.0.2 127.0.0.3 8081 PORT\r\n"))
		cc.Close()
	}()

	pc.readHeader()
	assert.Nil(t, pc.srcAddr)
	assert.Nil(t, pc.dstAddr)
	assert.Error(t, pc.readHeaderError)

	assert.NoError(t, l.Close())
}

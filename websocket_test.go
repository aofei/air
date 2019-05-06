package air

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketListen(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	buf := bytes.Buffer{}
	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		ws.TextHandler = func(text string) error {
			buf.WriteString(text)
			return nil
		}

		ws.BinaryHandler = func(b []byte) error {
			buf.Write(b)
			return nil
		}

		ws.ConnectionCloseHandler = func(
			status int,
			reason string,
		) error {
			buf.WriteString(fmt.Sprintf("%d - %s", status, reason))
			return errors.New(" - No Error")
		}

		ws.PingHandler = func(appData string) error {
			buf.WriteString(appData)
			return nil
		}

		ws.PongHandler = func(appData string) error {
			buf.WriteString(appData)
			return nil
		}

		ws.ErrorHandler = func(err error) {
			buf.WriteString(err.Error())
		}

		ws.Listen()
		ws.Listen() // Invalid call

		return nil
	})
	a.GET("/foo", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		ws.TextHandler = func(text string) error {
			return errors.New("Text Error")
		}

		ws.BinaryHandler = func(b []byte) error {
			return errors.New("Binary Error")
		}

		ws.ErrorHandler = func(err error) {
			buf.WriteString(err.Error())
		}

		ws.Listen()

		return nil
	})
	a.GET("/bar", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		ws.Listen()

		return nil
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	assert.NoError(t, conn.WriteMessage(
		websocket.TextMessage,
		[]byte("Foobar"),
	))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Foobar", buf.String())

	buf.Reset()

	assert.NoError(t, conn.WriteMessage(
		websocket.BinaryMessage,
		[]byte("Foobar"),
	))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Foobar", buf.String())

	buf.Reset()

	assert.NoError(t, conn.WriteMessage(
		websocket.PingMessage,
		[]byte("Foobar"),
	))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Foobar", buf.String())

	buf.Reset()

	assert.NoError(t, conn.WriteMessage(
		websocket.PongMessage,
		[]byte("Foobar"),
	))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Foobar", buf.String())

	buf.Reset()

	assert.NoError(t, conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(
			websocket.CloseNormalClosure,
			"Normal Closure",
		),
	))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "1000 - Normal Closure - No Error", buf.String())

	conn2, _, err := websocket.DefaultDialer.Dial(
		"ws://localhost:8080/foo",
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)
	defer conn2.Close()

	buf.Reset()

	assert.NoError(t, conn2.WriteMessage(websocket.TextMessage, nil))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Text Error", buf.String())

	buf.Reset()

	assert.NoError(t, conn2.WriteMessage(websocket.BinaryMessage, nil))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Binary Error", buf.String())

	conn3, _, err := websocket.DefaultDialer.Dial(
		"ws://localhost:8080/bar",
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, conn3)
	defer conn3.Close()

	buf.Reset()

	assert.NoError(t, conn3.WriteMessage(websocket.TextMessage, nil))
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, buf.String())

	buf.Reset()

	assert.NoError(t, conn3.WriteMessage(websocket.BinaryMessage, nil))
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, buf.String())
}

func TestWebSocketWriteText(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		return ws.WriteText("Foobar")
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	mt, m, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, mt)
	assert.Equal(t, []byte("Foobar"), m)
}

func TestWebSocketWriteBinary(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		return ws.WriteBinary([]byte("Foobar"))
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	mt, m, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, websocket.BinaryMessage, mt)
	assert.Equal(t, []byte("Foobar"), m)
}

func TestWebSocketWriteConnectionClose(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		return ws.WriteConnectionClose(
			websocket.CloseNormalClosure,
			"Normal Closure",
		)
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	buf := bytes.Buffer{}
	conn.SetCloseHandler(func(status int, reason string) error {
		buf.WriteString(fmt.Sprintf("%d - %s", status, reason))
		return nil
	})

	conn.ReadMessage()
	assert.Equal(t, "1000 - Normal Closure", buf.String())
}

func TestWebSocketWritePing(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		return ws.WritePing("Foobar")
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	buf := bytes.Buffer{}
	conn.SetPingHandler(func(appData string) error {
		buf.WriteString(appData)
		return nil
	})

	conn.ReadMessage()
	assert.Equal(t, "Foobar", buf.String())
}

func TestWebSocketWritePong(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}
		defer ws.Close()

		return ws.WritePong("Foobar")
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	buf := bytes.Buffer{}
	conn.SetPongHandler(func(appData string) error {
		buf.WriteString(appData)
		return nil
	})

	conn.ReadMessage()
	assert.Equal(t, "Foobar", buf.String())
}

func TestWebSocketClose(t *testing.T) {
	a := New()
	a.Address = "localhost:8080"

	a.GET("/", func(req *Request, res *Response) error {
		ws, err := res.WebSocket()
		if err != nil {
			return err
		}

		return ws.Close()
	})

	go a.Serve()
	defer a.Close()

	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080", nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer conn.Close()

	_, _, err = conn.ReadMessage()
	assert.True(t, websocket.IsCloseError(
		err,
		websocket.CloseAbnormalClosure,
	))
}

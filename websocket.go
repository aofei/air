package air

import (
	"io/ioutil"

	"github.com/gorilla/websocket"
)

// WebSocket is a WebSocket peer.
type WebSocket struct {
	TextHandler            func(text string) error
	BinaryHandler          func(b []byte) error
	ConnectionCloseHandler func(statusCode int, reason string) error
	PingHandler            func(appData string) error
	PongHandler            func(appData string) error
	ErrorHandler           func(err error)

	conn     *websocket.Conn
	listened bool
	closed   bool
}

// Listen listens for the messages sent from the remote peer of the ws.
func (ws *WebSocket) Listen() {
	if ws.listened {
		return
	}

	ws.listened = true

	go func() {
		for {
			if ws.closed {
				break
			}

			mt, r, err := ws.conn.NextReader()
			if err != nil {
				if !websocket.IsCloseError(
					err,
					websocket.CloseNormalClosure,
				) && ws.ErrorHandler != nil {
					ws.ErrorHandler(err)
				}

				break
			}

			switch mt {
			case websocket.TextMessage:
				if ws.TextHandler != nil {
					var b []byte
					b, err = ioutil.ReadAll(r)
					if err == nil {
						err = ws.TextHandler(string(b))
					}
				}
			case websocket.BinaryMessage:
				if ws.BinaryHandler != nil {
					var b []byte
					b, err = ioutil.ReadAll(r)
					if err == nil {
						err = ws.BinaryHandler(b)
					}
				}
			}

			if err != nil && ws.ErrorHandler != nil {
				ws.ErrorHandler(err)
			}
		}
	}()
}

// WriteText writes the text to the remote peer of the ws.
func (ws *WebSocket) WriteText(text string) error {
	return ws.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

// WriteBinary writes the b to the remote peer of the ws.
func (ws *WebSocket) WriteBinary(b []byte) error {
	return ws.conn.WriteMessage(websocket.BinaryMessage, b)
}

// WriteConnectionClose writes a connection close to the remote peer of the ws
// with the statusCode and the reason.
func (ws *WebSocket) WriteConnectionClose(statusCode int, reason string) error {
	return ws.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(statusCode, reason),
	)
}

// WritePing writes a ping to the remote peer of the ws with the appData.
func (ws *WebSocket) WritePing(appData string) error {
	return ws.conn.WriteMessage(websocket.PingMessage, []byte(appData))
}

// WritePong writes a pong to the remote peer of the ws with the appData.
func (ws *WebSocket) WritePong(appData string) error {
	return ws.conn.WriteMessage(websocket.PongMessage, []byte(appData))
}

// Close closes the ws without sending or waiting for a close message.
func (ws *WebSocket) Close() error {
	ws.closed = true
	return ws.conn.Close()
}

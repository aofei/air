package air

import (
	"io/ioutil"

	"github.com/gorilla/websocket"
)

// WebSocket is a WebSocket peer.
type WebSocket struct {
	// TextHandler is the handler that handles the incoming text messages of
	// the current WebSocket.
	TextHandler func(text string) error

	// BinaryHandler is the handler that handles the incoming binary
	// messages of the current WebSocket.
	BinaryHandler func(b []byte) error

	// ConnectionCloseHandler is the handler that handles the incoming
	// connection close messages of the current WebSocket.
	ConnectionCloseHandler func(status int, reason string) error

	// PingHandler is the handler that handles the incoming ping messages of
	// the current WebSocket.
	PingHandler func(appData string) error

	// PongHandler is the handler that handles the incoming pong messages of
	// the current WebSocket.
	PongHandler func(appData string) error

	// ErrorHandler is the handler that handles error occurs in the incoming
	// messages of the current WebSocket.
	ErrorHandler func(err error)

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
			if ws.TextHandler == nil {
				break
			}

			var b []byte
			if b, err = ioutil.ReadAll(r); err == nil {
				err = ws.TextHandler(string(b))
			}
		case websocket.BinaryMessage:
			if ws.BinaryHandler != nil {
				break
			}

			var b []byte
			if b, err = ioutil.ReadAll(r); err == nil {
				err = ws.BinaryHandler(b)
			}
		}

		if err != nil && ws.ErrorHandler != nil {
			ws.ErrorHandler(err)
		}
	}
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
// with the status and the reason.
func (ws *WebSocket) WriteConnectionClose(status int, reason string) error {
	return ws.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(status, reason),
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

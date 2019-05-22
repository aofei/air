package air

import (
	"io/ioutil"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket is a WebSocket peer.
//
// It is highly recommended not to modify the handlers of the `WebSocket` after
// calling the `WebSocket.Listen`, which will cause unpredictable problems.
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

	// Closed indicates whether the current WebSocket has been closed.
	Closed bool

	conn     *websocket.Conn
	listened bool
}

// SetMaxMessageBytes sets the maximum number of bytes the ws will read messages
// from the remote peer. If a message exceeds the limit, the ws sends a close
// message to the remote peer and returns an error immediately.
func (ws *WebSocket) SetMaxMessageBytes(mmb int64) {
	ws.conn.SetReadLimit(mmb)
}

// SetReadDeadline sets the read deadline on the underlying connection of the
// ws. After a read has timed out, the state of the ws is corrupt and all future
// reads will return an error immediately.
func (ws *WebSocket) SetReadDeadline(t time.Time) error {
	return ws.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection of the
// ws. After a write has timed out, the state of the ws is corrupt and all
// future writes will return an error immediately.
func (ws *WebSocket) SetWriteDeadline(t time.Time) error {
	return ws.conn.SetWriteDeadline(t)
}

// Listen listens for the messages sent from the remote peer of the ws. After
// one call to it, subsequent calls have no effect.
func (ws *WebSocket) Listen() {
	if ws.listened {
		return
	}

	ws.listened = true

	for {
		if ws.Closed {
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

			ws.Close() // Close it even if it has closed (insurance)

			continue
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
			if ws.BinaryHandler == nil {
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

// WriteText writes the text as a text message to the remote peer of the ws.
func (ws *WebSocket) WriteText(text string) error {
	return ws.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

// WriteBinary writes the b as a binary message to the remote peer of the ws.
func (ws *WebSocket) WriteBinary(b []byte) error {
	return ws.conn.WriteMessage(websocket.BinaryMessage, b)
}

// WriteConnectionClose writes a connection close message to the remote peer of
// the ws with the status and the reason.
func (ws *WebSocket) WriteConnectionClose(status int, reason string) error {
	return ws.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(status, reason),
	)
}

// WritePing writes a ping message to the remote peer of the ws with the
// appData.
func (ws *WebSocket) WritePing(appData string) error {
	return ws.conn.WriteMessage(websocket.PingMessage, []byte(appData))
}

// WritePong writes a pong message to the remote peer of the ws with the
// appData.
func (ws *WebSocket) WritePong(appData string) error {
	return ws.conn.WriteMessage(websocket.PongMessage, []byte(appData))
}

// Close closes the ws without sending or waiting for a close message.
func (ws *WebSocket) Close() error {
	ws.Closed = true
	return ws.conn.Close()
}

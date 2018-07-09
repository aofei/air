package air

import (
	"github.com/gorilla/websocket"
)

// WebSocketMessageType is the type of a WebSocket message.
//
// See RFC 6455, section 11.8.
type WebSocketMessageType uint8

// The WebSocket message types.
const (
	// WebSocketMessageTypeText is the type of a WebSocket message that
	// denotes a text data message. The text message payload is interpreted
	// as UTF-8 encoded text data.
	WebSocketMessageTypeText WebSocketMessageType = 1

	// WebSocketMessageTypeBinary is the type of a WebSocket message that
	// denotes a binary data message.
	WebSocketMessageTypeBinary WebSocketMessageType = 2

	// WebSocketMessageTypeConnectionClose is the type of a WebSocket
	// message that denotes a close control message. The optional message
	// payload contains a numeric code and text.
	WebSocketMessageTypeConnectionClose WebSocketMessageType = 8

	// WebSocketMessageTypePing is the type of a WebSocket message that
	// denotes a ping control message. The optional message payload is UTF-8
	// encoded text.
	WebSocketMessageTypePing WebSocketMessageType = 9

	// WebSocketMessageTypePong is the type of a WebSocket message that
	// denotes a pong control message. The optional message payload is UTF-8
	// encoded text.
	WebSocketMessageTypePong WebSocketMessageType = 10
)

// WebSocketConn is a WebSocket connection.
type WebSocketConn struct {
	ConnectionCloseHandler func(statusCode int, reason string) error
	PingHandler            func(appData string) error
	PongHandler            func(appData string) error

	conn *websocket.Conn
}

// Close closes the wsc without sending or waiting for a close message.
func (wsc *WebSocketConn) Close() error {
	return wsc.conn.Close()
}

// ReadMessage reads the next data message received from the peer of the wsc.
// The returned message's type is either `WebSocketMessageTypeText` or
// `WebSocketMessageTypeBinary`.
func (wsc *WebSocketConn) ReadMessage() (WebSocketMessageType, []byte, error) {
	t, b, err := wsc.conn.ReadMessage()
	return WebSocketMessageType(t), b, err
}

// WriteMessage writes the b to the peer of the wsc.
func (wsc *WebSocketConn) WriteMessage(t WebSocketMessageType, b []byte) error {
	return wsc.conn.WriteMessage(int(t), b)
}

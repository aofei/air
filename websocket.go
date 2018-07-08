package air

import "github.com/gorilla/websocket"

// WebSocketFrameType is the type of a WebSocket frame.
//
// See RFC 6455, section 11.8.
type WebSocketFrameType uint8

// The WebSocket frame types.
const (
	// WebSocketFrameTypeText is the type of a WebSocket frame that denotes
	// a text data frame. The text frame payload is interpreted as UTF-8
	// encoded text data.
	WebSocketFrameTypeText WebSocketFrameType = 1

	// WebSocketFrameTypeBinary is the type of a WebSocket frame that
	// denotes a binary data frame.
	WebSocketFrameTypeBinary WebSocketFrameType = 2

	// WebSocketFrameTypeConnectionClose is the type of a WebSocket frame
	// that denotes a close control frame. The optional frame payload
	// contains a numeric code and text.
	WebSocketFrameTypeConnectionClose WebSocketFrameType = 8

	// WebSocketFrameTypePing is the type of a WebSocket frame that denotes
	// a ping control frame. The optional frame payload is UTF-8 encoded
	// text.
	WebSocketFrameTypePing WebSocketFrameType = 9

	// WebSocketFrameTypePong is the type of a WebSocket frame that denotes
	// a pong control frame. The optional frame payload is UTF-8 encoded
	// text.
	WebSocketFrameTypePong WebSocketFrameType = 10
)

// WebSocketConn is a WebSocket connection.
type WebSocketConn struct {
	conn *websocket.Conn
}

// Close closes the underlying network connection without sending or waiting for
// a close frame.
func (wsc *WebSocketConn) Close() error {
	return wsc.conn.Close()
}

// ReadFrame reads the next data frame received from the peer. The returned type
// is either `WebSocketFrameTypeText` or `WebSocketFrameTypeBinary`.
func (wsc *WebSocketConn) ReadFrame() (WebSocketFrameType, []byte, error) {
	ft, b, err := wsc.conn.ReadMessage()
	return WebSocketFrameType(ft), b, err
}

// WriteFrame writes the b to the peer.
func (wsc *WebSocketConn) WriteFrame(wsft WebSocketFrameType, b []byte) error {
	return wsc.conn.WriteMessage(int(wsft), b)
}

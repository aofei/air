package air

import "github.com/gorilla/websocket"

// WebSocketCloseStatusCode is the status code numbers of the WebSocket
// connection close.
//
// See RFC 6455, section 11.7.
type WebSocketCloseStatusCode uint16

// The WebSocket close status codes.
const (
	// WebSocketCloseStatusCodeNormalClosure indicates a normal closure,
	// meaning that the purpose for which the connection was established
	// has been fulfilled.
	WebSocketCloseStatusCodeNormalClosure = 1000

	// WebSocketCloseStatusCodeGoingAway indicates that an endpoint is
	// "going away", such as a server going down or a browser having
	// navigated away from a page.
	WebSocketCloseStatusCodeGoingAway = 1001

	// WebSocketCloseStatusCodeProtocolError indicates that an endpoint is
	// terminating the connection due to a protocol error.
	WebSocketCloseStatusCodeProtocolError = 1002

	// WebSocketCloseStatusCodeUnsupportedData indicates that an endpoint is
	// terminating the connection because it has received a type of data it
	// cannot accept (e.g., an endpoint that understands only text data MAY
	// send this if it receives a binary message).
	WebSocketCloseStatusCodeUnsupportedData = 1003

	// WebSocketCloseStatusCodeNoStatusReceived is a reserved value and MUST
	// NOT be set as a status code in a Close control frame by an endpoint.
	// It is designated for use in applications expecting a status code to
	// indicate that no status code was actually present.
	WebSocketCloseStatusCodeNoStatusReceived = 1005

	// WebSocketCloseStatusCodeAbnormalClosure is a reserved value and MUST
	// NOT be set as a status code in a Close control frame by an endpoint.
	// It is designated for use in applications expecting a status code to
	// indicate that the connection was closed abnormally, e.g., without
	// sending or receiving a Close control frame.
	WebSocketCloseStatusCodeAbnormalClosure = 1006

	// WebSocketCloseStatusCodeInvalidFramePayloadData indicates that an
	// endpoint is terminating the connection because it has received data
	// within a message that was not consistent with the type of the message
	// (e.g., non-UTF-8 [RFC 3629] data within a text message).
	WebSocketCloseStatusCodeInvalidFramePayloadData = 1007

	// WebSocketCloseStatusCodePolicyViolation indicates that an endpoint is
	// terminating the connection because it has received a message that
	// violates its policy. This is a generic status code that can be
	// returned when there is no other more suitable status code (e.g., 1003
	// or 1009) or if there is a need to hide specific details about the
	// policy.
	WebSocketCloseStatusCodePolicyViolation = 1008

	// WebSocketCloseStatusCodeMessageTooBig indicates that an endpoint is
	// terminating the connection because it has received a message that is
	// too big for it to process.
	WebSocketCloseStatusCodeMessageTooBig = 1009

	// WebSocketCloseStatusCodeMandatoryExtension indicates that an endpoint
	// (client) is terminating the connection because it has expected the
	// server to negotiate one or more extension, but the server did not
	// return them in the response message of the WebSocket handshake. The
	// list of extensions that are needed SHOULD appear in the /reason/ part
	// of the Close frame. Note that this status code is not used by the
	// server, because it can fail the WebSocket handshake instead.
	WebSocketCloseStatusCodeMandatoryExtension = 1010

	// WebSocketCloseStatusCodeInternalServerError indicates that a server
	// is terminating the connection because it encountered an unexpected
	// condition that prevented it from fulfilling the request.
	WebSocketCloseStatusCodeInternalServerError = 1011

	// WebSocketCloseStatusCodeTLSHandshake is a reserved value and MUST NOT
	// be set as a status code in a Close control frame by an endpoint. It
	// is designated for use in applications expecting a status code to
	// indicate that the connection was closed due to a failure to perform a
	// TLS handshake (e.g., the server certificate cannot be verified).
	WebSocketCloseStatusCodeTLSHandshake = 1015
)

// WebSocketMessageType is the type of a WebSocket message.
//
// See RFC 6455, section 11.8.
type WebSocketMessageType uint8

// The WebSocket message types.
const (
	// WebSocketMessageTypeContinuation is the type of a WebSocket message
	// that denotes a continuation message.
	WebSocketMessageTypeContinuation WebSocketMessageType = 0

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

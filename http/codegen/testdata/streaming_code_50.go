package testdata


var StreamingPayloadPrimitiveArrayClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection and
// reads instances of "[]string" from the connection.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) CloseAndRecv() ([]string, error) {
	var (
		rv   []string
		body []string
		err  error
	)
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection and
// reads instances of "[]string" from the connection with context.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) CloseAndRecvWithContext(ctx context.Context) ([]string, error) {
	return s.CloseAndRecv()
}
`



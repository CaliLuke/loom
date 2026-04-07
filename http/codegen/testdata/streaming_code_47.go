package testdata


var StreamingPayloadPrimitiveArrayServerStreamSendCode = `// SendAndClose streams instances of "[]string" to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection and
// closes the connection.
func (s *StreamingPayloadPrimitiveArrayMethodServerStream) SendAndClose(v []string) error {
	defer s.conn.Close()
	res := v
	return s.conn.WriteJSON(res)
}

// SendAndCloseWithContext streams instances of "[]string" to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadPrimitiveArrayMethodServerStream) SendAndCloseWithContext(ctx context.Context, v []string) error {
	return s.SendAndClose(v)
}
`



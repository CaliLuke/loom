package testdata


var StreamingPayloadPrimitiveServerStreamSendCode = `// SendAndCloseWithContext streams instances of "string" to the
// "StreamingPayloadPrimitiveMethod" endpoint websocket connection with context
// and closes the connection.
func (s *StreamingPayloadPrimitiveMethodServerStream) SendAndCloseWithContext(ctx context.Context, v string) error {
	defer s.conn.Close()
	res := v
	return s.conn.WriteJSON(res)
}

// SendAndClose streams instances of "string" to the
// "StreamingPayloadPrimitiveMethod" endpoint websocket connection and closes
// the connection.
func (s *StreamingPayloadPrimitiveMethodServerStream) SendAndClose(v string) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
`



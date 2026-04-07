package testdata


var StreamingPayloadResultWithExplicitViewClientStreamSendCode = `// Send streams instances of "float32" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithExplicitViewMethodClientStream) Send(v float32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "float32" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// with context.
func (s *StreamingPayloadResultWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v float32) error {
	return s.Send(v)
}
`



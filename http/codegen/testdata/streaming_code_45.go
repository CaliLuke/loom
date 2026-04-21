package testdata


var StreamingPayloadPrimitiveClientStreamSendCode = `// SendWithContext streams instances of "string" to the
// "StreamingPayloadPrimitiveMethod" endpoint websocket connection with context.
func (s *StreamingPayloadPrimitiveMethodClientStream) SendWithContext(ctx context.Context, v string) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "string" to the "StreamingPayloadPrimitiveMethod"
// endpoint websocket connection.
func (s *StreamingPayloadPrimitiveMethodClientStream) Send(v string) error {
	return s.SendWithContext(context.Background(), v)
}
`



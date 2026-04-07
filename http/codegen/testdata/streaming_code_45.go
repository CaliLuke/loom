package testdata


var StreamingPayloadPrimitiveClientStreamSendCode = `// Send streams instances of "string" to the "StreamingPayloadPrimitiveMethod"
// endpoint websocket connection.
func (s *StreamingPayloadPrimitiveMethodClientStream) Send(v string) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "string" to the
// "StreamingPayloadPrimitiveMethod" endpoint websocket connection with context.
func (s *StreamingPayloadPrimitiveMethodClientStream) SendWithContext(ctx context.Context, v string) error {
	return s.Send(v)
}
`



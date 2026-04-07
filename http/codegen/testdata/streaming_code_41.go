package testdata


var StreamingPayloadResultCollectionWithExplicitViewClientStreamSendCode = `// Send streams instances of "any" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) Send(v any) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "any" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.Send(v)
}
`



package testdata


var StreamingPayloadResultCollectionWithExplicitViewClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "any" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`



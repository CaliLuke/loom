package testdata


var StreamingPayloadResultCollectionWithViewsClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "any" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`



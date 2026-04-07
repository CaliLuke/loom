package testdata


var StreamingPayloadResultCollectionWithViewsClientStreamSendCode = `// Send streams instances of "any" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) Send(v any) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "any" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.Send(v)
}
`



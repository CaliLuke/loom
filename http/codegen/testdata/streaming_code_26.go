package testdata


var StreamingPayloadResultWithViewsClientStreamSendCode = `// Send streams instances of "float32" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithViewsMethodClientStream) Send(v float32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "float32" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadResultWithViewsMethodClientStream) SendWithContext(ctx context.Context, v float32) error {
	return s.Send(v)
}
`



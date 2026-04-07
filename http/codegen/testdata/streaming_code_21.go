package testdata


var StreamingPayloadNoResultClientStreamSendCode = `// Send streams instances of "string" to the "StreamingPayloadNoResultMethod"
// endpoint websocket connection.
func (s *StreamingPayloadNoResultMethodClientStream) Send(v string) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "string" to the
// "StreamingPayloadNoResultMethod" endpoint websocket connection with context.
func (s *StreamingPayloadNoResultMethodClientStream) SendWithContext(ctx context.Context, v string) error {
	return s.Send(v)
}
`



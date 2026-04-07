package testdata


var StreamingPayloadClientStreamSendCode = `// Send streams instances of "streamingpayloadservice.Request" to the
// "StreamingPayloadMethod" endpoint websocket connection.
func (s *StreamingPayloadMethodClientStream) Send(v *streamingpayloadservice.Request) error {
	body := NewStreamingPayloadMethodStreamingBody(v)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of "streamingpayloadservice.Request" to
// the "StreamingPayloadMethod" endpoint websocket connection with context.
func (s *StreamingPayloadMethodClientStream) SendWithContext(ctx context.Context, v *streamingpayloadservice.Request) error {
	return s.Send(v)
}
`



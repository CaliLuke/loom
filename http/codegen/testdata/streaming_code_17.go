package testdata


var StreamingPayloadNoPayloadClientStreamSendCode = `// SendWithContext streams instances of
// "streamingpayloadnopayloadservice.Request" to the
// "StreamingPayloadNoPayloadMethod" endpoint websocket connection with context.
func (s *StreamingPayloadNoPayloadMethodClientStream) SendWithContext(ctx context.Context, v *streamingpayloadnopayloadservice.Request) error {
	body := NewStreamingPayloadNoPayloadMethodStreamingBody(v)
	return s.conn.WriteJSON(body)
}

// Send streams instances of "streamingpayloadnopayloadservice.Request" to the
// "StreamingPayloadNoPayloadMethod" endpoint websocket connection.
func (s *StreamingPayloadNoPayloadMethodClientStream) Send(v *streamingpayloadnopayloadservice.Request) error {
	return s.SendWithContext(context.Background(), v)
}
`



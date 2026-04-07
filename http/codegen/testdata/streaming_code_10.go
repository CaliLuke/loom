package testdata


var StreamingPayloadServerStreamSendCode = `// SendAndClose streams instances of "streamingpayloadservice.UserType" to the
// "StreamingPayloadMethod" endpoint websocket connection and closes the
// connection.
func (s *StreamingPayloadMethodServerStream) SendAndClose(v *streamingpayloadservice.UserType) error {
	defer s.conn.Close()
	res := v
	body := NewStreamingPayloadMethodResponseBody(res)
	return s.conn.WriteJSON(body)
}

// SendAndCloseWithContext streams instances of
// "streamingpayloadservice.UserType" to the "StreamingPayloadMethod" endpoint
// websocket connection with context and closes the connection.
func (s *StreamingPayloadMethodServerStream) SendAndCloseWithContext(ctx context.Context, v *streamingpayloadservice.UserType) error {
	return s.SendAndClose(v)
}
`



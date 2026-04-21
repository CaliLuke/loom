package testdata


var StreamingPayloadUserTypeArrayServerStreamSendCode = `// SendAndCloseWithContext streams instances of "string" to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadUserTypeArrayMethodServerStream) SendAndCloseWithContext(ctx context.Context, v string) error {
	defer s.conn.Close()
	res := v
	return s.conn.WriteJSON(res)
}

// SendAndClose streams instances of "string" to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection and
// closes the connection.
func (s *StreamingPayloadUserTypeArrayMethodServerStream) SendAndClose(v string) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
`



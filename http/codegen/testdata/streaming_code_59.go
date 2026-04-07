package testdata


var StreamingPayloadUserTypeMapServerStreamSendCode = `// SendAndClose streams instances of "[]string" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection and closes
// the connection.
func (s *StreamingPayloadUserTypeMapMethodServerStream) SendAndClose(v []string) error {
	defer s.conn.Close()
	res := v
	return s.conn.WriteJSON(res)
}

// SendAndCloseWithContext streams instances of "[]string" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadUserTypeMapMethodServerStream) SendAndCloseWithContext(ctx context.Context, v []string) error {
	return s.SendAndClose(v)
}
`



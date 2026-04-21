package testdata


var StreamingPayloadPrimitiveMapServerStreamSendCode = `// SendAndCloseWithContext streams instances of "map[int]int" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadPrimitiveMapMethodServerStream) SendAndCloseWithContext(ctx context.Context, v map[int]int) error {
	defer s.conn.Close()
	res := v
	return s.conn.WriteJSON(res)
}

// SendAndClose streams instances of "map[int]int" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection and
// closes the connection.
func (s *StreamingPayloadPrimitiveMapMethodServerStream) SendAndClose(v map[int]int) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
`



package testdata


var StreamingPayloadPrimitiveMapClientStreamSendCode = `// Send streams instances of "map[string]int32" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection.
func (s *StreamingPayloadPrimitiveMapMethodClientStream) Send(v map[string]int32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "map[string]int32" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadPrimitiveMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]int32) error {
	return s.Send(v)
}
`



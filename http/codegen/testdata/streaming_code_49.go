package testdata


var StreamingPayloadPrimitiveArrayClientStreamSendCode = `// Send streams instances of "[]int32" to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) Send(v []int32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "[]int32" to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) SendWithContext(ctx context.Context, v []int32) error {
	return s.Send(v)
}
`



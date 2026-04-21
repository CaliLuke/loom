package testdata


var StreamingPayloadPrimitiveArrayClientStreamSendCode = `// SendWithContext streams instances of "[]int32" to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) SendWithContext(ctx context.Context, v []int32) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "[]int32" to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) Send(v []int32) error {
	return s.SendWithContext(context.Background(), v)
}
`



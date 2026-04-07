package testdata


var BidirectionalStreamingPrimitiveArrayClientStreamSendCode = `// Send streams instances of "[]int32" to the
// "BidirectionalStreamingPrimitiveArrayMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveArrayMethodClientStream) Send(v []int32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "[]int32" to the
// "BidirectionalStreamingPrimitiveArrayMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingPrimitiveArrayMethodClientStream) SendWithContext(ctx context.Context, v []int32) error {
	return s.Send(v)
}
`



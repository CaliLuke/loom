package testdata


var BidirectionalStreamingPrimitiveMapClientStreamSendCode = `// SendWithContext streams instances of "map[string]int32" to the
// "BidirectionalStreamingPrimitiveMapMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingPrimitiveMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]int32) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "map[string]int32" to the
// "BidirectionalStreamingPrimitiveMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveMapMethodClientStream) Send(v map[string]int32) error {
	return s.SendWithContext(context.Background(), v)
}
`



package testdata


var BidirectionalStreamingPrimitiveClientStreamSendCode = `// SendWithContext streams instances of "string" to the
// "BidirectionalStreamingPrimitiveMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingPrimitiveMethodClientStream) SendWithContext(ctx context.Context, v string) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "string" to the
// "BidirectionalStreamingPrimitiveMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveMethodClientStream) Send(v string) error {
	return s.SendWithContext(context.Background(), v)
}
`



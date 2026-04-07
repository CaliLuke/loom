package testdata


var BidirectionalStreamingResultWithExplicitViewClientStreamSendCode = `// Send streams instances of "float32" to the
// "BidirectionalStreamingResultWithExplicitViewMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultWithExplicitViewMethodClientStream) Send(v float32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "float32" to the
// "BidirectionalStreamingResultWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v float32) error {
	return s.Send(v)
}
`



package testdata


var BidirectionalStreamingResultWithViewsClientStreamSendCode = `// Send streams instances of "float32" to the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) Send(v float32) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "float32" to the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) SendWithContext(ctx context.Context, v float32) error {
	return s.Send(v)
}
`



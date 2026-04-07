package testdata


var BidirectionalStreamingResultCollectionWithViewsClientStreamSendCode = `// Send streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) Send(v any) error {
	return s.conn.WriteJSON(v)
}

// SendWithContext streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.Send(v)
}
`



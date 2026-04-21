package testdata


var BidirectionalStreamingResultCollectionWithViewsClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`



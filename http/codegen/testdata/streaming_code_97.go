package testdata


var BidirectionalStreamingResultCollectionWithExplicitViewClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithExplicitViewMethod" endpoint
// websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v any) error {
	return s.conn.WriteJSON(v)
}

// Send streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithExplicitViewMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`



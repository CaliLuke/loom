package testdata

var BidirectionalStreamingResultWithViewsClientStreamCloseCode = `// Close closes the "BidirectionalStreamingResultWithViewsMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) Close() error {
	return s.conn.CloseWithFinalJSON(nil)
}
`

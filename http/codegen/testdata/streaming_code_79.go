package testdata

var BidirectionalStreamingResultWithViewsServerStreamCloseCode = `// Close closes the "BidirectionalStreamingResultWithViewsMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodServerStream) Close() error {
	return s.conn.CloseWithStatus(websocket.CloseNormalClosure, "")
}
`

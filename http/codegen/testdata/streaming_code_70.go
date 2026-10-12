package testdata

var BidirectionalStreamingClientStreamCloseCode = `// Close closes the "BidirectionalStreamingMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingMethodClientStream) Close() error {
	return s.conn.CloseWithFinalJSON(nil)
}
`

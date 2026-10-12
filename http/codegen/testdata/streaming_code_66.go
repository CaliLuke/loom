package testdata

var BidirectionalStreamingServerStreamCloseCode = `// Close closes the "BidirectionalStreamingMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingMethodServerStream) Close() error {
	return s.conn.CloseWithStatus(websocket.CloseNormalClosure, "")
}
`

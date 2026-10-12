package testdata

var BidirectionalStreamingNoPayloadServerStreamCloseCode = `// Close closes the "BidirectionalStreamingNoPayloadMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingNoPayloadMethodServerStream) Close() error {
	return s.conn.CloseWithStatus(websocket.CloseNormalClosure, "")
}
`

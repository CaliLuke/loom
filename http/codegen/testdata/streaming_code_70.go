package testdata

var BidirectionalStreamingClientStreamCloseCode = `// Close closes the "BidirectionalStreamingMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingMethodClientStream) Close() error {
	var err error
	// Send a nil payload to the server implying client closing connection.
	if err = s.conn.WriteJSON(context.Background(), nil); err != nil {
		return err
	}
	return s.conn.Close()
}
`

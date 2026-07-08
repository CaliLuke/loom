package testdata

var BidirectionalStreamingResultWithViewsClientStreamCloseCode = `// Close closes the "BidirectionalStreamingResultWithViewsMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) Close() error {
	var err error
	// Send a nil payload to the server implying client closing connection.
	if err = s.conn.WriteJSON(context.Background(), nil); err != nil {
		return err
	}
	return s.conn.Close()
}
`

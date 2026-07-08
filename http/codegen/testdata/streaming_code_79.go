package testdata

var BidirectionalStreamingResultWithViewsServerStreamCloseCode = `// Close closes the "BidirectionalStreamingResultWithViewsMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodServerStream) Close() error {
	var err error
	if s.conn == nil {
		return nil
	}
	if err = s.conn.WriteClose("server closing connection"); err != nil {
		return err
	}
	return s.conn.Close()
}
`

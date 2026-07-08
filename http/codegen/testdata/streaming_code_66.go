package testdata

var BidirectionalStreamingServerStreamCloseCode = `// Close closes the "BidirectionalStreamingMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingMethodServerStream) Close() error {
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

package testdata

var BidirectionalStreamingNoPayloadServerStreamCloseCode = `// Close closes the "BidirectionalStreamingNoPayloadMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingNoPayloadMethodServerStream) Close() error {
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

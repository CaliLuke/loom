package testdata


var BidirectionalStreamingNoPayloadClientStreamCloseCode = `// Close closes the "BidirectionalStreamingNoPayloadMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) Close() error {
	var err error
	// Send a nil payload to the server implying client closing connection.
	if err = s.conn.WriteJSON(nil); err != nil {
		return err
	}
	return s.conn.Close()
}
`



package testdata


var StreamingPayloadNoResultClientStreamCloseCode = `// Close closes the "StreamingPayloadNoResultMethod" endpoint websocket
// connection.
func (s *StreamingPayloadNoResultMethodClientStream) Close() error {
	var err error
	// Send a nil payload to the server implying client closing connection.
	if err = s.conn.WriteJSON(nil); err != nil {
		return err
	}
	return s.conn.Close()
}
`



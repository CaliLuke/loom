package testdata

var StreamingPayloadNoResultClientStreamCloseCode = `// Close closes the "StreamingPayloadNoResultMethod" endpoint websocket
// connection.
func (s *StreamingPayloadNoResultMethodClientStream) Close() error {
	return s.conn.CloseWithFinalJSON(nil)
}
`

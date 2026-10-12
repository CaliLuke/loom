package testdata

var StreamingPayloadNoResultServerStreamCloseCode = `// Close closes the "StreamingPayloadNoResultMethod" endpoint websocket
// connection.
func (s *StreamingPayloadNoResultMethodServerStream) Close() error {
	return s.conn.CloseWithStatus(websocket.CloseNormalClosure, "")
}
`

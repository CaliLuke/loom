package testdata


var BidirectionalStreamingPrimitiveClientStreamRecvCode = `// Recv reads instances of "string" from the
// "BidirectionalStreamingPrimitiveMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveMethodClientStream) Recv() (string, error) {
	var (
		rv   string
		body string
		err  error
	)
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// RecvWithContext reads instances of "string" from the
// "BidirectionalStreamingPrimitiveMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingPrimitiveMethodClientStream) RecvWithContext(ctx context.Context) (string, error) {
	return s.Recv()
}
`



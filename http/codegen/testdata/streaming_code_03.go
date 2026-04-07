package testdata


var StreamingResultPrimitiveMapClientStreamRecvCode = `// Recv reads instances of "map[int32]string" from the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection.
func (s *StreamingResultPrimitiveMapMethodClientStream) Recv() (map[int32]string, error) {
	var (
		rv   map[int32]string
		body map[int32]string
		err  error
	)
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// RecvWithContext reads instances of "map[int32]string" from the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection with
// context.
func (s *StreamingResultPrimitiveMapMethodClientStream) RecvWithContext(ctx context.Context) (map[int32]string, error) {
	return s.Recv()
}
`



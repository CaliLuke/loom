package testdata

var BidirectionalStreamingPrimitiveMapClientStreamRecvCode = `// Recv reads instances of "map[int]int" from the
// "BidirectionalStreamingPrimitiveMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveMapMethodClientStream) Recv() (map[int]int, error) {
	var (
		rv   map[int]int
		body map[int]int
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// RecvWithContext reads instances of "map[int]int" from the
// "BidirectionalStreamingPrimitiveMapMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingPrimitiveMapMethodClientStream) RecvWithContext(ctx context.Context) (map[int]int, error) {
	var (
		rv   map[int]int
		body map[int]int
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}
`

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
	var rv string
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	stopContextWatch := context.AfterFunc(ctx, func() {
		if s.conn == nil {
			return
		}
		if closeErr := s.conn.Close(); closeErr != nil {
			return
		}
	})
	defer stopContextWatch()
	v, err := s.Recv()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
	}
	return v, err
}
`



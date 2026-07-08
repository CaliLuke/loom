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
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
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
	var rv map[int32]string
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

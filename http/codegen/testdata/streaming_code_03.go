package testdata

var StreamingResultPrimitiveMapClientStreamRecvCode = `// Recv reads instances of "map[int32]string" from the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection.
func (s *StreamingResultPrimitiveMapMethodClientStream) Recv() (map[int32]string, error) {
	var (
		rv   map[int32]string
		body map[int32]loom.Nullable[string]
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
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
	for _, v := range body {
		if _, ok := v.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullMapValueError("body[key]"))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultPrimitiveMapMethodMapInt32StringOK(body)
	return res, nil
}

// RecvWithContext reads instances of "map[int32]string" from the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection with
// context.
func (s *StreamingResultPrimitiveMapMethodClientStream) RecvWithContext(ctx context.Context) (map[int32]string, error) {
	var (
		rv   map[int32]string
		body map[int32]loom.Nullable[string]
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
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
	for _, v := range body {
		if _, ok := v.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullMapValueError("body[key]"))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultPrimitiveMapMethodMapInt32StringOK(body)
	return res, nil
}
`

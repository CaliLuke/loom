package testdata

var StreamingPayloadPrimitiveArrayServerStreamRecvCode = `// Recv reads instances of "[]int32" from the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection.
func (s *StreamingPayloadPrimitiveArrayMethodServerStream) Recv() ([]int32, error) {
	return s.RecvWithContext(s.r.Context())
}

// RecvWithContext reads instances of "[]int32" from the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadPrimitiveArrayMethodServerStream) RecvWithContext(ctx context.Context) ([]int32, error) {
	var (
		rv   []int32
		body []loom.Nullable[int32]
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	// Upgrade the HTTP connection to a websocket connection only once. Connection
	// upgrade is done here so that authorization logic in the endpoint is executed
	// before calling the actual service method which may call Recv().
	s.once.Do(func() {
		var conn *websocket.Conn
		conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
		if err != nil {
			s.upgradeErr = err
			return
		}
		if s.configurer != nil {
			conn = s.configurer(conn, s.cancel)
		}
		s.conn.SetConn(conn)
		if err = ctx.Err(); err != nil {
			if closeErr := s.conn.Close(); closeErr != nil {
				s.upgradeErr = closeErr
				return
			}
			s.upgradeErr = err
			return
		}
	})
	if s.upgradeErr != nil {
		return rv, s.upgradeErr
	}
	if err = s.conn.ReadJSON(ctx, &body); err != nil {
		return rv, err
	}
	if body == nil {
		return rv, io.EOF
	}
	for i, e := range body {
		if _, ok := e.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	return NewStreamingPayloadPrimitiveArrayMethodArray(body), nil
}
`

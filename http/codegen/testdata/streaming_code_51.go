package testdata


var StreamingPayloadPrimitiveMapServerStreamSendCode = `// SendAndCloseWithContext streams instances of "map[int]int" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadPrimitiveMapMethodServerStream) SendAndCloseWithContext(ctx context.Context, v map[int]int) error {
	if err := ctx.Err(); err != nil {
		return err
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
	err := func() error {
		defer s.conn.Close()
		res := v
		return s.conn.WriteJSON(res)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// SendAndClose streams instances of "map[int]int" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection and
// closes the connection.
func (s *StreamingPayloadPrimitiveMapMethodServerStream) SendAndClose(v map[int]int) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`



package testdata

var StreamingPayloadUserTypeMapServerStreamSendCode = `// SendAndCloseWithContext streams instances of "[]string" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadUserTypeMapMethodServerStream) SendAndCloseWithContext(ctx context.Context, v []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		defer s.conn.Close()
		res := v
		return s.conn.WriteJSON(ctx, res)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// SendAndClose streams instances of "[]string" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection and closes
// the connection.
func (s *StreamingPayloadUserTypeMapMethodServerStream) SendAndClose(v []string) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`

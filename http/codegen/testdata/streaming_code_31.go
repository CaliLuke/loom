package testdata

var StreamingPayloadResultWithExplicitViewClientStreamSendCode = `// SendWithContext streams instances of "float32" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// with context.
func (s *StreamingPayloadResultWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v float32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		return s.conn.WriteJSON(ctx, v)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "float32" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithExplicitViewMethodClientStream) Send(v float32) error {
	return s.SendWithContext(context.Background(), v)
}
`

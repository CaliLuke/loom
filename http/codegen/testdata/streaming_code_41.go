package testdata

var StreamingPayloadResultCollectionWithExplicitViewClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v any) error {
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

// Send streams instances of "any" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`

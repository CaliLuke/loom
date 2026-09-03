package testdata

var StreamingPayloadResultCollectionWithExplicitViewClientStreamSendCode = `// SendWithContext streams instances of "loom.JSONValue" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v loom.JSONValue) error {
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

// Send streams instances of "loom.JSONValue" to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) Send(v loom.JSONValue) error {
	return s.SendWithContext(context.Background(), v)
}
`

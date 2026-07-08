package testdata

var StreamingPayloadResultCollectionWithViewsClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) SendWithContext(ctx context.Context, v any) error {
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
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`

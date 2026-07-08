package testdata

var StreamingPayloadResultWithViewsClientStreamSendCode = `// SendWithContext streams instances of "float32" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadResultWithViewsMethodClientStream) SendWithContext(ctx context.Context, v float32) error {
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
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithViewsMethodClientStream) Send(v float32) error {
	return s.SendWithContext(context.Background(), v)
}
`

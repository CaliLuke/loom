package testdata

var StreamingPayloadNoResultClientStreamSendCode = `// SendWithContext streams instances of "string" to the
// "StreamingPayloadNoResultMethod" endpoint websocket connection with context.
func (s *StreamingPayloadNoResultMethodClientStream) SendWithContext(ctx context.Context, v string) error {
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

// Send streams instances of "string" to the "StreamingPayloadNoResultMethod"
// endpoint websocket connection.
func (s *StreamingPayloadNoResultMethodClientStream) Send(v string) error {
	return s.SendWithContext(context.Background(), v)
}
`

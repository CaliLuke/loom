package testdata

var StreamingPayloadClientStreamSendCode = `// SendWithContext streams instances of "streamingpayloadservice.Request" to
// the "StreamingPayloadMethod" endpoint websocket connection with context.
func (s *StreamingPayloadMethodClientStream) SendWithContext(ctx context.Context, v *streamingpayloadservice.Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		body := NewStreamingPayloadMethodStreamingBody(v)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "streamingpayloadservice.Request" to the
// "StreamingPayloadMethod" endpoint websocket connection.
func (s *StreamingPayloadMethodClientStream) Send(v *streamingpayloadservice.Request) error {
	return s.SendWithContext(context.Background(), v)
}
`

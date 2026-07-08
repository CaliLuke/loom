package testdata

var StreamingPayloadNoPayloadClientStreamSendCode = `// SendWithContext streams instances of
// "streamingpayloadnopayloadservice.Request" to the
// "StreamingPayloadNoPayloadMethod" endpoint websocket connection with context.
func (s *StreamingPayloadNoPayloadMethodClientStream) SendWithContext(ctx context.Context, v *streamingpayloadnopayloadservice.Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		body := NewStreamingPayloadNoPayloadMethodStreamingBody(v)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "streamingpayloadnopayloadservice.Request" to the
// "StreamingPayloadNoPayloadMethod" endpoint websocket connection.
func (s *StreamingPayloadNoPayloadMethodClientStream) Send(v *streamingpayloadnopayloadservice.Request) error {
	return s.SendWithContext(context.Background(), v)
}
`

package testdata


var StreamingPayloadClientStreamSendCode = `// SendWithContext streams instances of "streamingpayloadservice.Request" to
// the "StreamingPayloadMethod" endpoint websocket connection with context.
func (s *StreamingPayloadMethodClientStream) SendWithContext(ctx context.Context, v *streamingpayloadservice.Request) error {
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
		body := NewStreamingPayloadMethodStreamingBody(v)
		return s.conn.WriteJSON(body)
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



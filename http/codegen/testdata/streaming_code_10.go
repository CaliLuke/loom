package testdata

var StreamingPayloadServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadservice.UserType" to the "StreamingPayloadMethod" endpoint
// websocket connection with context and closes the connection.
func (s *StreamingPayloadMethodServerStream) SendAndCloseWithContext(ctx context.Context, v *streamingpayloadservice.UserType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		defer s.conn.Close()
		res := v
		body := NewStreamingPayloadMethodResponseBody(res)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// SendAndClose streams instances of "streamingpayloadservice.UserType" to the
// "StreamingPayloadMethod" endpoint websocket connection and closes the
// connection.
func (s *StreamingPayloadMethodServerStream) SendAndClose(v *streamingpayloadservice.UserType) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`

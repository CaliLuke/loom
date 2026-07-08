package testdata

var StreamingPayloadUserTypeArrayClientStreamSendCode = `// SendWithContext streams instances of
// "[]*streamingpayloadusertypearrayservice.RequestType" to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadUserTypeArrayMethodClientStream) SendWithContext(ctx context.Context, v []*streamingpayloadusertypearrayservice.RequestType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		body := NewRequestType(v)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "[]*streamingpayloadusertypearrayservice.RequestType" to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection.
func (s *StreamingPayloadUserTypeArrayMethodClientStream) Send(v []*streamingpayloadusertypearrayservice.RequestType) error {
	return s.SendWithContext(context.Background(), v)
}
`

package testdata

var StreamingPayloadUserTypeMapClientStreamSendCode = `// SendWithContext streams instances of
// "map[string]*streamingpayloadusertypemapservice.RequestType" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadUserTypeMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]*streamingpayloadusertypemapservice.RequestType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		body := NewMapStringRequestType(v)
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
// "map[string]*streamingpayloadusertypemapservice.RequestType" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection.
func (s *StreamingPayloadUserTypeMapMethodClientStream) Send(v map[string]*streamingpayloadusertypemapservice.RequestType) error {
	return s.SendWithContext(context.Background(), v)
}
`

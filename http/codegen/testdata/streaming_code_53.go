package testdata

var StreamingPayloadPrimitiveMapClientStreamSendCode = `// SendWithContext streams instances of "map[string]int32" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadPrimitiveMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]int32) error {
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

// Send streams instances of "map[string]int32" to the
// "StreamingPayloadPrimitiveMapMethod" endpoint websocket connection.
func (s *StreamingPayloadPrimitiveMapMethodClientStream) Send(v map[string]int32) error {
	return s.SendWithContext(context.Background(), v)
}
`

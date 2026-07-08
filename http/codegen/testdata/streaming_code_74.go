package testdata

var BidirectionalStreamingNoPayloadClientStreamSendCode = `// SendWithContext streams instances of
// "bidirectionalstreamingnopayloadservice.Request" to the
// "BidirectionalStreamingNoPayloadMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) SendWithContext(ctx context.Context, v *bidirectionalstreamingnopayloadservice.Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		body := NewBidirectionalStreamingNoPayloadMethodStreamingBody(v)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "bidirectionalstreamingnopayloadservice.Request"
// to the "BidirectionalStreamingNoPayloadMethod" endpoint websocket connection.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) Send(v *bidirectionalstreamingnopayloadservice.Request) error {
	return s.SendWithContext(context.Background(), v)
}
`

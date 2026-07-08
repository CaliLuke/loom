package testdata

var BidirectionalStreamingNoPayloadClientStreamRecvCode = `// Recv reads instances of "bidirectionalstreamingnopayloadservice.UserType"
// from the "BidirectionalStreamingNoPayloadMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) Recv() (*bidirectionalstreamingnopayloadservice.UserType, error) {
	var (
		rv   *bidirectionalstreamingnopayloadservice.UserType
		body BidirectionalStreamingNoPayloadMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingNoPayloadMethodUserTypeOK(&body)
	return res, nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingnopayloadservice.UserType" from the
// "BidirectionalStreamingNoPayloadMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) RecvWithContext(ctx context.Context) (*bidirectionalstreamingnopayloadservice.UserType, error) {
	var (
		rv   *bidirectionalstreamingnopayloadservice.UserType
		body BidirectionalStreamingNoPayloadMethodResponseBody
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingNoPayloadMethodUserTypeOK(&body)
	return res, nil
}
`

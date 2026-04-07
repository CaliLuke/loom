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
	err = s.conn.ReadJSON(&body)
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
	return s.Recv()
}
`



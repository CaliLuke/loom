package testdata

var BidirectionalStreamingClientStreamRecvCode = `// Recv reads instances of "bidirectionalstreamingservice.UserType" from the
// "BidirectionalStreamingMethod" endpoint websocket connection.
func (s *BidirectionalStreamingMethodClientStream) Recv() (*bidirectionalstreamingservice.UserType, error) {
	var (
		rv   *bidirectionalstreamingservice.UserType
		body BidirectionalStreamingMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingMethodUserTypeOK(&body)
	return res, nil
}

// RecvWithContext reads instances of "bidirectionalstreamingservice.UserType"
// from the "BidirectionalStreamingMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingMethodClientStream) RecvWithContext(ctx context.Context) (*bidirectionalstreamingservice.UserType, error) {
	var (
		rv   *bidirectionalstreamingservice.UserType
		body BidirectionalStreamingMethodResponseBody
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
	res := NewBidirectionalStreamingMethodUserTypeOK(&body)
	return res, nil
}
`

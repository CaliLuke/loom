package testdata


var BidirectionalStreamingResultWithExplicitViewClientStreamRecvCode = `// Recv reads instances of
// "bidirectionalstreamingresultwithexplicitviewservice.Usertype" from the
// "BidirectionalStreamingResultWithExplicitViewMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultWithExplicitViewMethodClientStream) Recv() (*bidirectionalstreamingresultwithexplicitviewservice.Usertype, error) {
	var (
		rv   *bidirectionalstreamingresultwithexplicitviewservice.Usertype
		body BidirectionalStreamingResultWithExplicitViewMethodResponseBodyExtended
		err  error
	)
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingResultWithExplicitViewMethodUsertypeOK(&body)
	vres := &bidirectionalstreamingresultwithexplicitviewserviceviews.Usertype{res, "extended"}
	if err := bidirectionalstreamingresultwithexplicitviewserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultWithExplicitViewService", "BidirectionalStreamingResultWithExplicitViewMethod", err)
	}
	return bidirectionalstreamingresultwithexplicitviewservice.NewUsertype(vres), nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingresultwithexplicitviewservice.Usertype" from the
// "BidirectionalStreamingResultWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultWithExplicitViewMethodClientStream) RecvWithContext(ctx context.Context) (*bidirectionalstreamingresultwithexplicitviewservice.Usertype, error) {
	return s.Recv()
}
`



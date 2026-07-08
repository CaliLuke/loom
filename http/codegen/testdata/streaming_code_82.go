package testdata

var BidirectionalStreamingResultWithViewsClientStreamRecvCode = `// Recv reads instances of
// "bidirectionalstreamingresultwithviewsservice.Usertype" from the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) Recv() (*bidirectionalstreamingresultwithviewsservice.Usertype, error) {
	var (
		rv   *bidirectionalstreamingresultwithviewsservice.Usertype
		body BidirectionalStreamingResultWithViewsMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingResultWithViewsMethodUsertypeOK(&body)
	vres := &bidirectionalstreamingresultwithviewsserviceviews.Usertype{res, s.view}
	if err := bidirectionalstreamingresultwithviewsserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultWithViewsService", "BidirectionalStreamingResultWithViewsMethod", err)
	}
	result, err := bidirectionalstreamingresultwithviewsservice.NewUsertype(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultWithViewsService", "BidirectionalStreamingResultWithViewsMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingresultwithviewsservice.Usertype" from the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) RecvWithContext(ctx context.Context) (*bidirectionalstreamingresultwithviewsservice.Usertype, error) {
	var rv *bidirectionalstreamingresultwithviewsservice.Usertype
	if err := ctx.Err(); err != nil {
		return rv, err
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
	v, err := s.Recv()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
	}
	return v, err
}
`

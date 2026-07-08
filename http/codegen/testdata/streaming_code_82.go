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
	err = s.conn.ReadJSON(context.Background(), &body)
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
	var (
		rv   *bidirectionalstreamingresultwithviewsservice.Usertype
		body BidirectionalStreamingResultWithViewsMethodResponseBody
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
`

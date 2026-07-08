package testdata

var BidirectionalStreamingResultCollectionWithViewsClientStreamRecvCode = `// Recv reads instances of
// "bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection"
// from the "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) Recv() (bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection, error) {
	var (
		rv   bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection
		body BidirectionalStreamingResultCollectionWithViewsMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingResultCollectionWithViewsMethodUsertypeCollectionOK(body)
	vres := bidirectionalstreamingresultcollectionwithviewsserviceviews.UsertypeCollection{res, s.view}
	if err := bidirectionalstreamingresultcollectionwithviewsserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithViewsService", "BidirectionalStreamingResultCollectionWithViewsMethod", err)
	}
	result, err := bidirectionalstreamingresultcollectionwithviewsservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithViewsService", "BidirectionalStreamingResultCollectionWithViewsMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection"
// from the "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint
// websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) RecvWithContext(ctx context.Context) (bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection, error) {
	var (
		rv   bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection
		body BidirectionalStreamingResultCollectionWithViewsMethodResponseBody
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
	res := NewBidirectionalStreamingResultCollectionWithViewsMethodUsertypeCollectionOK(body)
	vres := bidirectionalstreamingresultcollectionwithviewsserviceviews.UsertypeCollection{res, s.view}
	if err := bidirectionalstreamingresultcollectionwithviewsserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithViewsService", "BidirectionalStreamingResultCollectionWithViewsMethod", err)
	}
	result, err := bidirectionalstreamingresultcollectionwithviewsservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithViewsService", "BidirectionalStreamingResultCollectionWithViewsMethod", err)
	}
	return result, nil
}
`

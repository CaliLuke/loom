package testdata

var BidirectionalStreamingResultCollectionWithExplicitViewClientStreamRecvCode = `// Recv reads instances of
// "bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection"
// from the "BidirectionalStreamingResultCollectionWithExplicitViewMethod"
// endpoint websocket connection.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) Recv() (bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	var (
		rv   bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection
		body UsertypeResponseTinyCollection
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingResultCollectionWithExplicitViewMethodUsertypeCollectionOK(body)
	vres := bidirectionalstreamingresultcollectionwithexplicitviewserviceviews.UsertypeCollection{res, "tiny"}
	if err := bidirectionalstreamingresultcollectionwithexplicitviewserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithExplicitViewService", "BidirectionalStreamingResultCollectionWithExplicitViewMethod", err)
	}
	result, err := bidirectionalstreamingresultcollectionwithexplicitviewservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithExplicitViewService", "BidirectionalStreamingResultCollectionWithExplicitViewMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection"
// from the "BidirectionalStreamingResultCollectionWithExplicitViewMethod"
// endpoint websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) RecvWithContext(ctx context.Context) (bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	var (
		rv   bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection
		body UsertypeResponseTinyCollection
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
	res := NewBidirectionalStreamingResultCollectionWithExplicitViewMethodUsertypeCollectionOK(body)
	vres := bidirectionalstreamingresultcollectionwithexplicitviewserviceviews.UsertypeCollection{res, "tiny"}
	if err := bidirectionalstreamingresultcollectionwithexplicitviewserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithExplicitViewService", "BidirectionalStreamingResultCollectionWithExplicitViewMethod", err)
	}
	result, err := bidirectionalstreamingresultcollectionwithexplicitviewservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("BidirectionalStreamingResultCollectionWithExplicitViewService", "BidirectionalStreamingResultCollectionWithExplicitViewMethod", err)
	}
	return result, nil
}
`

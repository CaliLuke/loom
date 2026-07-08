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
	err = s.conn.ReadJSON(&body)
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
	return bidirectionalstreamingresultcollectionwithviewsservice.NewUsertypeCollection(vres), nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection"
// from the "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint
// websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) RecvWithContext(ctx context.Context) (bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection, error) {
	var rv bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection
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



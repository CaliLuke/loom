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
	err = s.conn.ReadJSON(&body)
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
	return bidirectionalstreamingresultcollectionwithexplicitviewservice.NewUsertypeCollection(vres), nil
}

// RecvWithContext reads instances of
// "bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection"
// from the "BidirectionalStreamingResultCollectionWithExplicitViewMethod"
// endpoint websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) RecvWithContext(ctx context.Context) (bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	var rv bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection
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



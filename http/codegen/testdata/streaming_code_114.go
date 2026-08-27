package testdata

var BidirectionalStreamingUserTypeArrayClientStreamRecvCode = `// Recv reads instances of
// "[]*bidirectionalstreamingusertypearrayservice.ResultType" from the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeArrayMethodClientStream) Recv() ([]*bidirectionalstreamingusertypearrayservice.ResultType, error) {
	var (
		rv   []*bidirectionalstreamingusertypearrayservice.ResultType
		body []*ResultTypeResponse
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	for i, e := range body {
		if e == nil {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingUserTypeArrayMethodResultTypeOK(body)
	return res, nil
}

// RecvWithContext reads instances of
// "[]*bidirectionalstreamingusertypearrayservice.ResultType" from the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingUserTypeArrayMethodClientStream) RecvWithContext(ctx context.Context) ([]*bidirectionalstreamingusertypearrayservice.ResultType, error) {
	var (
		rv   []*bidirectionalstreamingusertypearrayservice.ResultType
		body []*ResultTypeResponse
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
	for i, e := range body {
		if e == nil {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingUserTypeArrayMethodResultTypeOK(body)
	return res, nil
}
`

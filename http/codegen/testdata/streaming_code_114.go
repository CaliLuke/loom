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
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
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
	var rv []*bidirectionalstreamingusertypearrayservice.ResultType
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



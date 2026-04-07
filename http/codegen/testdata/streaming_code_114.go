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
	return s.Recv()
}
`



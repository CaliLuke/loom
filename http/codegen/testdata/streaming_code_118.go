package testdata


var BidirectionalStreamingUserTypeMapClientStreamRecvCode = `// Recv reads instances of
// "map[string]*bidirectionalstreamingusertypemapservice.ResultType" from the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeMapMethodClientStream) Recv() (map[string]*bidirectionalstreamingusertypemapservice.ResultType, error) {
	var (
		rv   map[string]*bidirectionalstreamingusertypemapservice.ResultType
		body map[string]*ResultTypeResponse
		err  error
	)
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewBidirectionalStreamingUserTypeMapMethodMapStringResultTypeOK(body)
	return res, nil
}

// RecvWithContext reads instances of
// "map[string]*bidirectionalstreamingusertypemapservice.ResultType" from the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingUserTypeMapMethodClientStream) RecvWithContext(ctx context.Context) (map[string]*bidirectionalstreamingusertypemapservice.ResultType, error) {
	var rv map[string]*bidirectionalstreamingusertypemapservice.ResultType
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

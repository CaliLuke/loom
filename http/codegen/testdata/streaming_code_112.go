package testdata

var BidirectionalStreamingUserTypeArrayServerStreamRecvCode = `// Recv reads instances of
// "[]*bidirectionalstreamingusertypearrayservice.RequestType" from the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeArrayMethodServerStream) Recv() ([]*bidirectionalstreamingusertypearrayservice.RequestType, error) {
	return s.RecvWithContext(s.r.Context())
}

// RecvWithContext reads instances of
// "[]*bidirectionalstreamingusertypearrayservice.RequestType" from the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingUserTypeArrayMethodServerStream) RecvWithContext(ctx context.Context) ([]*bidirectionalstreamingusertypearrayservice.RequestType, error) {
	var (
		rv   []*bidirectionalstreamingusertypearrayservice.RequestType
		body []loom.Nullable[*RequestType]
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	// Upgrade the HTTP connection to a websocket connection only once. Connection
	// upgrade is done here so that authorization logic in the endpoint is executed
	// before calling the actual service method which may call Recv().
	s.once.Do(func() {
		var conn *websocket.Conn
		conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
		if err != nil {
			s.upgradeErr = err
			return
		}
		if s.configurer != nil {
			conn = s.configurer(conn, s.cancel)
		}
		s.conn.SetConn(conn)
		if err = ctx.Err(); err != nil {
			if closeErr := s.conn.Close(); closeErr != nil {
				s.upgradeErr = closeErr
				return
			}
			s.upgradeErr = err
			return
		}
	})
	if s.upgradeErr != nil {
		return rv, s.upgradeErr
	}
	if err = s.conn.ReadJSON(ctx, &body); err != nil {
		return rv, err
	}
	if body == nil {
		return rv, io.EOF
	}
	for i, e := range body {
		if _, ok := e.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	return NewBidirectionalStreamingUserTypeArrayMethodArray(body), nil
}
`

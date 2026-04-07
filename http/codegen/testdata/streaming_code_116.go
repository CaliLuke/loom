package testdata


var BidirectionalStreamingUserTypeMapServerStreamRecvCode = `// Recv reads instances of
// "map[string]*bidirectionalstreamingusertypemapservice.RequestType" from the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeMapMethodServerStream) Recv() (map[string]*bidirectionalstreamingusertypemapservice.RequestType, error) {
	var (
		rv   map[string]*bidirectionalstreamingusertypemapservice.RequestType
		body map[string]*RequestType
		err  error
	)
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
		s.conn = conn
	})
	if s.upgradeErr != nil {
		return rv, s.upgradeErr
	}
	if err = s.conn.ReadJSON(&body); err != nil {
		return rv, err
	}
	if body == nil {
		return rv, io.EOF
	}
	return NewBidirectionalStreamingUserTypeMapMethodMap(body), nil
}

// RecvWithContext reads instances of
// "map[string]*bidirectionalstreamingusertypemapservice.RequestType" from the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingUserTypeMapMethodServerStream) RecvWithContext(ctx context.Context) (map[string]*bidirectionalstreamingusertypemapservice.RequestType, error) {
	return s.Recv()
}
`



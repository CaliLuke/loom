package testdata


var BidirectionalStreamingResultCollectionWithViewsServerStreamRecvCode = `// Recv reads instances of "any" from the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) Recv() (any, error) {
	var (
		rv  any
		msg *any
		err error
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
	if err = s.conn.ReadJSON(&msg); err != nil {
		return rv, err
	}
	if msg == nil {
		return rv, io.EOF
	}
	return *msg, nil
}

// RecvWithContext reads instances of "any" from the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) RecvWithContext(ctx context.Context) (any, error) {
	return s.Recv()
}
`



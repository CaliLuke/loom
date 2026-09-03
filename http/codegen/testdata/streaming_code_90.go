package testdata

var BidirectionalStreamingResultCollectionWithViewsServerStreamRecvCode = `// Recv reads instances of "loom.JSONValue" from the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) Recv() (loom.JSONValue, error) {
	return s.RecvWithContext(s.r.Context())
}

// RecvWithContext reads instances of "loom.JSONValue" from the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) RecvWithContext(ctx context.Context) (loom.JSONValue, error) {
	var (
		rv  loom.JSONValue
		msg *loom.JSONValue
		err error
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
	if err = s.conn.ReadJSON(ctx, &msg); err != nil {
		return rv, err
	}
	if msg == nil {
		return rv, io.EOF
	}
	return *msg, nil
}
`

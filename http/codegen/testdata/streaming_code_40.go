package testdata

var StreamingPayloadResultCollectionWithExplicitViewServerStreamRecvCode = `// Recv reads instances of "any" from the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodServerStream) Recv() (any, error) {
	return s.RecvWithContext(s.r.Context())
}

// RecvWithContext reads instances of "any" from the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodServerStream) RecvWithContext(ctx context.Context) (any, error) {
	var (
		rv  any
		msg *any
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

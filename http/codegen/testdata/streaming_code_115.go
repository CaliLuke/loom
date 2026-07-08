package testdata

var BidirectionalStreamingUserTypeMapServerStreamSendCode = `// SendWithContext streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.ResultType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingUserTypeMapMethodServerStream) SendWithContext(ctx context.Context, v map[string]*bidirectionalstreamingusertypemapservice.ResultType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
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
			return s.upgradeErr
		}
		res := v
		body := NewBidirectionalStreamingUserTypeMapMethodResponseBody(res)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.ResultType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeMapMethodServerStream) Send(v map[string]*bidirectionalstreamingusertypemapservice.ResultType) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

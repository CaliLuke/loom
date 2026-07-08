package testdata


var BidirectionalStreamingServerStreamRecvCode = `// Recv reads instances of "bidirectionalstreamingservice.Request" from the
// "BidirectionalStreamingMethod" endpoint websocket connection.
func (s *BidirectionalStreamingMethodServerStream) Recv() (*bidirectionalstreamingservice.Request, error) {
	return s.RecvWithContext(s.r.Context())
}

// RecvWithContext reads instances of "bidirectionalstreamingservice.Request"
// from the "BidirectionalStreamingMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingMethodServerStream) RecvWithContext(ctx context.Context) (*bidirectionalstreamingservice.Request, error) {
	var (
		rv  *bidirectionalstreamingservice.Request
		msg *BidirectionalStreamingMethodStreamingBody
		err error
	)
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
	if err = s.conn.ReadJSON(&msg); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
		return rv, err
	}
	if msg == nil {
		return rv, io.EOF
	}
	body := *msg
	err = ValidateBidirectionalStreamingMethodStreamingBody(&body)
	if err != nil {
		return rv, err
	}
	return NewBidirectionalStreamingMethodStreamingBody(msg), nil
}
`

